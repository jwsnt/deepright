package ai.deepright.llm.summary;

import ai.deepright.llm.provider.RequestProviderUtils;
import ai.open.right.WorkflowException;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryTruncate;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import ai.open.right.workflow.flow.summary.impl.SummaryServiceImpl;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.nio.ByteBuffer;
import java.nio.CharBuffer;
import java.nio.charset.CharsetEncoder;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

@Slf4j
@Getter
@Setter
public class SchemaSummary extends SummaryServiceImpl {

    public static final String NAME = "summary_service";

    protected Integer truncate;

    @Override
    protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
        return History.buildMarkdown(histories, SummaryTruncate.builder()
                .summaryConfig(summaryConfig)
                .truncate(this.truncate)
                .workTask(workTask)
                .build());
    }

    protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content, int retry) throws Exception {
        // 自定义解析
        try {
            Summary[] summaries = JsonUtils.array(content) ? JsonUtils.read(content, Summary[].class) : JsonUtils.transfer(MapUtils.getObject(JsonUtils.read(content, Map.class), "items"), Summary[].class);
            if (ArrayUtils.getLength(summaries) < 2 && log.isWarnEnabled()) {
                // 可能归纳产生了异常
                log.warn("The summary less than 2 histories, content={}", StringUtils.abbreviate(content, Byte.MAX_VALUE));
            }
            return Arrays.stream(Summary.toPairs(summaries, workTask)).toList();
        } catch (Exception e) {
            if (JsonUtils.map(content) && retry == 0) {
                // 尝试转为[]（xiaomi）
                return this.buildPairs(summaryConfig, workTask, "[" + content + "]", retry + 1);
            }
            if (log.isErrorEnabled()) {
                log.error("The summary failed={}", content, e);
            }
            Summary summary = new Summary();
            summary.setType(Summary.TYPE_ANSWER);
            summary.setContent(content);
            return Arrays.stream(Summary.toPairs(new Summary[]{summary}, workTask)).toList();
        }
    }

    @Override
    protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
        return this.buildPairs(summaryConfig, workTask, content, 0);
    }

    @Builder
    @Getter
    public static class SummaryTruncate implements HistoryTruncate {

        protected SummaryConfig summaryConfig;

        protected WorkflowTask workTask;

        protected Integer truncate;

        @Override
        public String truncate(String histories) throws Exception {
            if (!RequestProviderUtils.isMultiInputModel(this.workTask) && BytesUtils.utf8Bytes(histories) > this.truncate) {
                CharsetEncoder encoder = StandardCharsets.UTF_8.newEncoder();
                ByteBuffer buffer = ByteBuffer.allocate(this.truncate);
                encoder.encode(CharBuffer.wrap(histories.toCharArray()), buffer, true);
                buffer.flip();
                return StandardCharsets.UTF_8.decode(buffer).toString();
            } else {
                return histories;
            }
        }
    }

    @Getter
    @Setter
    public static class Summary {

        public static final String TYPE_ANSWER = "ANSWER";

        public static final String TYPE_QUERY = "QUERY";

        protected Long created;

        protected String content;

        protected String type;

        public HistoryPair toPair() {
            HistoryPair pair = new HistoryPair();
            pair.setCreated(this.created);
            pair.setFunction(History.FUN_CHAT);
            if (StringUtils.startsWithIgnoreCase(Summary.TYPE_ANSWER, this.type)) {
                pair.setRole(History.ROLE_ASSISTANT);
                pair.setAnswer(this.content);
            } else {
                pair.setRole(History.ROLE_USER);
                pair.setQuery(this.content);
            }
            return pair;
        }

        public static HistoryPair[] toPairs(Summary[] summaries, WorkflowTask workTask) {
            WorkflowException.checkCondition(ArrayUtils.isEmpty(summaries), "The history summaries must not be empty");
            HistoryPair[] pairs = new HistoryPair[summaries.length];
            for (int i = 0; i < summaries.length; i++) {
                pairs[i] = summaries[i].toPair();
            }
            return pairs;
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class SchemaServiceConfig extends SummaryServiceImpl.InitConfig {

        @Value("${summary.truncate:128000}")
        protected Integer truncate;

        @Override
        @Bean(SchemaSummary.NAME)
        @ConditionalOnMissingBean(name = SchemaSummary.NAME)
        public SchemaSummary summaryService() throws Exception {
            SchemaSummary schemaSummary = new SchemaSummary();
            BeanUtils.copyProperties(this, schemaSummary);
            schemaSummary.setMaxSize(this.summary4maxSize != null ? this.summary4maxSize : (int) (this.provider4maxSize * this.provider4maxRate));
            log.info("SchemaSummary inited, timeout4Llm={}", schemaSummary.getTimeout4Llm());
            return schemaSummary;
        }
    }
}
