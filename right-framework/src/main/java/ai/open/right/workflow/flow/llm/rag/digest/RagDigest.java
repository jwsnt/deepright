package ai.open.right.workflow.flow.llm.rag.digest;

import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.XmlUtils;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAsync;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.store.digest.Digest;
import ai.open.right.workflow.flow.llm.store.digest.DigestStore;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.StringReader;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;

@Slf4j
@Setter
@Getter
// 使用摘要记忆增强内容
public class RagDigest extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_digest";

    protected ExecutorService executorService;

    protected DigestStore digestStore;

    // Rag Digest（摘要记忆）调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        return super.allowed(ragConfig, ragData) && ragConfig.hasRagDigest();
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        Assert.notEmpty(ragConfig.getRagDigestConfig().getKeys(), "Digest can not be empty");
        if (log.isInfoEnabled()) {
            log.info("Rag digest start");
        }
        return new RagAsync(ragConfig, this.executorService.submit(new DigestFuture(ragConfig, ragData)), this.timeout4Llm);
    }

    public class DigestFuture implements Callable<Void> {

        protected final RagConfig ragConfig;

        protected final RagData ragData;

        public DigestFuture(RagConfig ragConfig, RagData ragData) {
            this.ragConfig = ragConfig;
            this.ragData = ragData;
        }

        @Override
        public Void call() throws Exception {
            SyncConfig syncConfig = SyncConfig.builder()
                    .workflow(this.ragConfig.getRagDigestConfig().getDynamic())
                    .timeout(RagDigest.this.timeout4Llm)
                    .workTask(this.ragData.getQuery())
                    .build();
            String digestBody = SyncWorkflowTask.exeWorkflow(RagDigest.this.getNotifierService(), syncConfig).get();
            Assert.hasText(digestBody, "Digest can not be empty");
            Map<String, Object> ragRequest = this.parseDigest(digestBody);
            String ragResponse = this.upsertDigest(this.ragConfig.getRagDigestConfig(), new Digest(ragRequest, this.ragConfig.getRagDigestConfig().getKeys()));
            if (log.isInfoEnabled()) {
                log.info("Digest={}-{}", ragRequest, ragResponse);
            }
            RagService.updatePrompt(this.ragConfig, this.ragData, this.ragConfig.getReplace(), ragResponse);
            return null;
        }

        // 更新或插入Digest
        protected String upsertDigest(RagDigestConfig ragDigestConfig, Digest digest) throws Exception {
            Digest current = RagDigest.this.digestStore.upsert(this.ragData.getQuery(), this.ragConfig.getRagDigestConfig().getScene(this.ragData.getQuery().getWorkflow()), digest);
            Assert.notNull(current, "Current digest can not be empty");
            String content = null;
            if (current.hasDigest()) {
                if (ragDigestConfig.isMode(RagDigestConfig.MODE_JSON)) {
                    // 使用JSON解析
                    content = JsonUtils.write(new DigestData(current.getDigest()));
                } else {
                    // 使用XML解析
                    content = XmlUtils.write(new DigestData(current.getDigest()));
                }
            }
            if (log.isInfoEnabled()) {
                log.info("Upsert digest={}", digest);
            }
            return content;
        }

        protected Map<String, Object> parseDigest(String digest) throws Exception {
            if (log.isDebugEnabled()) {
                log.debug("The digest to be parsed={}", digest);
            }
            if (JsonUtils.like(digest)) {
                // 使用JSON解析
                Map<String, Object> jsonDigest = JsonUtils.read(digest, Map.class);
                if (log.isDebugEnabled()) {
                    log.debug("The digest parsed into JSON={}", jsonDigest);
                }
                return jsonDigest;
            } else {
                // 使用文本行解析
                Map<String, Object> textDigests = new HashMap<String, Object>();
                for (String line : IOUtils.readLines(new StringReader(digest))) {
                    String[] pair = line.split("=");
                    textDigests.put(pair[0], pair.length == 2 ? pair[1] : org.apache.commons.lang3.StringUtils.join(Arrays.copyOfRange(pair, 1, pair.length), "="));
                }
                if (log.isDebugEnabled()) {
                    log.debug("The digest parsed into Text={}", textDigests);
                }
                return textDigests;
            }
        }
    }

    @JacksonXmlRootElement(localName = "Digest")
    public static class DigestData extends HashMap<String, Object> {

        public DigestData(Map<String, Object> values) {
            this.putAll(values);
        }
    }

    @ConditionalOnProperty(name = "digest.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected DigestStore digestStore;

        @Value("${digest.timeout.llm:1800000}")
        // Rag Digest（摘要记忆）调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Bean(RagDigest.RAG_KEY)
        @ConditionalOnMissingBean(name = RagDigest.RAG_KEY)
        public RagDigest ragDigest() throws Exception {
            RagDigest ragDigest = new RagDigest();
            BeanUtils.copyProperties(this, ragDigest);
            log.info("RagDigest inited: timeout4Llm={},timeout4Condition={}", ragDigest.getTimeout4Llm(), ragDigest.getTimeout4Condition());
            return ragDigest;
        }
    }

}
