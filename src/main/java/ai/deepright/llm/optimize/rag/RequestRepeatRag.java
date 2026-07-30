package ai.deepright.llm.optimize.rag;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.provider.RequestContextBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.llm.LLMFunCallData;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.notify.Notifier;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

@Slf4j
@Getter
@Setter
public class RequestRepeatRag extends RagCondition implements RagService {

    public static final String LANG_KEY_REQUEST_REPEAT_FOOTER = "request.repeat.footer";

    public static final String RAG_KEY = "rag_repeat";

    protected ResourceService resourceService;

    protected String template4repeat;

    protected Integer offset;

    @PostConstruct
    public void init() throws Exception {
        this.template4repeat = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4repeat).openStream()), StandardCharsets.UTF_8);
        WorkflowException.checkCondition(StringUtils.isEmpty(this.template4repeat), "The template repeat must not be empty");
        WorkflowException.checkCondition(this.offset == null || this.offset < 2, "The repeat offset must be at least 2");
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.checkFunCallRepeat(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    @Override
    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        // 跳过测试请求
        return super.allowed(ragConfig, ragData) && !FeatureFlag.isTest(ragData.getQuery());
    }

    protected void checkFunCallRepeat(RagConfig ragConfig, RagData ragData) throws Exception {
        LLMFunCallData funCallData = ragData.getRequest().getFunCallData();
        if (funCallData == null) {
            return;
        }
        // 如果Response content或Request reason连续 N 条相同文本假定为循环
        if (this.isResponseRepeat(funCallData, ragConfig, ragData) || this.isRequestRepeat(funCallData, ragConfig, ragData)) {
            this.buildHistoryRepeat(ragConfig, ragData);
            this.notifyFooter(ragConfig, ragData);
        }
    }

    protected boolean isResponseRepeat(LLMFunCallData funCallData, RagConfig ragConfig, RagData ragData) throws Exception {
        List<LLMFunCallResponse> responses = funCallData.getResponses();
        if (CollectionUtils.size(responses) < this.offset) {
            return false;
        }
        String lastMessage = responses.getLast().getResponse();
        for (int i = responses.size() - 2; i >= responses.size() - this.offset; i--) {
            LLMFunCallResponse response = responses.get(i);
            if (!StringUtils.equalsIgnoreCase(lastMessage, response.getResponse())) {
                return false;
            }
        }
        return true;
    }

    protected boolean isRequestRepeat(LLMFunCallData funCallData, RagConfig ragConfig, RagData ragData) throws Exception {
        List<LLMFunCallRequest> requests = funCallData.getRequests();
        if (CollectionUtils.size(requests) < this.offset) {
            return false;
        }
        String lastMessage = requests.getLast().getReason();
        // 需要包含空格排除
        if (StringUtils.isBlank(lastMessage)) {
            return false;
        }
        for (int i = requests.size() - 2; i >= requests.size() - this.offset; i--) {
            LLMFunCallRequest request = requests.get(i);
            if (!StringUtils.equalsIgnoreCase(lastMessage, request.getReason())) {
                return false;
            }
        }
        return true;
    }

    protected void buildHistoryRepeat(RagConfig ragConfig, RagData ragData) throws Exception {
        // 在当前时间（最后插入提醒）
        List<History> histories = ragData.getQuery().getHistories();
        histories = histories != null ? histories : new ArrayList<History>();
        // Role=User
        histories.add(RequestContextBuilder.buildContext(ragData.getRequest(), this.template4repeat));
        ragData.getQuery().setHistories(histories);
    }

    protected void notifyFooter(RagConfig ragConfig, RagData ragData) throws Exception {
        this.notifierService.notify(Segment.build(ragData.getQuery(), Segment.SegmentConfig.builder()
                .content(new StringBuffer(XmlResourceLang.get(RequestRepeatRag.LANG_KEY_REQUEST_REPEAT_FOOTER)))
                .metadata(CliPrinter.process(RequestExpiredRag.RAG_KEY))
                .workflow(ragData.getQuery().getWorkflow())
                .notifier(Notifier.SOURCE)
                .build()), ragData.getQuery(), ragData.getQuery());
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Value("${optimize.repeat.template:classpath:config/optimize/repeat.md}")
        protected String template4repeat;

        @Value("${optimize.repeat.offset:5}")
        protected Integer offset;

        @Bean(RequestRepeatRag.RAG_KEY)
        @ConditionalOnMissingBean(name = RequestRepeatRag.RAG_KEY)
        public RequestRepeatRag requestRepeatRag() throws Exception {
            RequestRepeatRag requestRepeatRag = new RequestRepeatRag();
            BeanUtils.copyProperties(this, requestRepeatRag);
            log.info("RequestRepeatRag inited, timeout4Condition={}", requestRepeatRag.getTimeout4Condition());
            return requestRepeatRag;
        }
    }
}
