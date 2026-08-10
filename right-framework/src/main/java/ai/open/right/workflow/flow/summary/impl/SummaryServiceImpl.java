package ai.open.right.workflow.flow.summary.impl;

import ai.open.right.WorkflowException;
import ai.open.right.utils.BytesUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.condition.ConditionUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestModel;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.summary.SummaryConfig;
import ai.open.right.workflow.flow.summary.SummaryPart;
import ai.open.right.workflow.flow.summary.SummaryService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.fasterxml.jackson.core.JacksonException;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlElementWrapper;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlProperty;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlText;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Base64;
import java.util.List;
import java.util.stream.Collectors;

@Slf4j
@Setter
@Getter
public class SummaryServiceImpl implements SummaryService {

    protected NotifierService notifierService;

    protected HistoryStore historyStore;

    // 摘要调用下游思考链（Workflow）的超时
    protected Integer timeout4Llm;

    protected DefStore defStore;

    // LLM最大长度
    protected Integer maxSize;

    protected String mimeType;

    protected String requery;

    protected String store;

    @Override
    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
        try {
            // 过滤
            histories = this.selectHistories(summaryConfig, workTask, this.updateHistories(summaryConfig, histories));
            if (CollectionUtils.isEmpty(histories) || !this.allowed(summaryConfig, workTask, histories, append)) {
                return null;
            }
            if (log.isDebugEnabled()) {
                log.debug("The summary restore history size={}", histories.size());
            }
            String query = this.buildQuery(summaryConfig, workTask, histories, append);
            List<MediaContext> mediaContexts = this.buildMediaContext(summaryConfig, workTask, histories, query);
            Assert.hasText(summaryConfig.getDynamic(), "The summary dynamic can not be empty");
            SyncConfig syncConfig = SyncConfig.builder()
                    // 如果存在MediaContext则改写Query
                    .reQuery(this.buildReQuery(summaryConfig, workTask, mediaContexts, query))
                    .timeout(summaryConfig.getTimeout4Llm(this.timeout4Llm))
                    .workflow(summaryConfig.getDynamic())
                    .mediaContext(mediaContexts)
                    .workTask(workTask)
                    .build();
            String content = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isDebugEnabled()) {
                log.debug("The summary content={}", content);
            }
            SummaryPart summaryPart = this.buildSummaryPart(summaryConfig, workTask, content).init(workTask, History.buildLastTimeline(histories));
            if (summaryConfig.shouldStore() && !CollectionUtils.isEmpty(summaryPart.getPairs())) {
                Long lastTime = this.buildLastTimeline(summaryConfig, workTask, histories);
                // 更新并重置
                summaryPart = this.update(summaryConfig, workTask, summaryPart, lastTime);
                summaryPart = this.reset(summaryConfig, workTask, summaryPart, lastTime);
            }
            return summaryPart;
        } catch (Exception e) {
            this.dropOnFailed(summaryConfig, workTask, histories);
            throw e;
        }
    }

    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        return this.summarize(summaryConfig, workTask, histories, "");
    }

    @Override
    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask, String append) throws Exception {
        Assert.notNull(this.historyStore, "The history store can not be empty");
        return this.summarize(summaryConfig, workTask, this.historyStore.restore(workTask, summaryConfig.getScene(workTask.getWorkflow()), summaryConfig.getMaxsize(), summaryConfig.getDesc(), (summaryConfig.getNow() != null ? -summaryConfig.getNow() : null)), append);
    }

    @Override
    public SummaryPart summarize(SummaryConfig summaryConfig, WorkflowTask workTask) throws Exception {
        return this.summarize(summaryConfig, workTask, "");
    }

    protected List<MediaContext> buildMediaContext(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String requery) throws Exception {
        if (StringUtils.isEmpty(this.store) || !this.defStore.supportFunction(this.store)) {
            if (log.isDebugEnabled()) {
                log.debug("The summary store can not support media context={}", this.store);
            }
            return null;
        }
        FileStore fileStore = this.defStore.fetchStore(this.store);
        int size = BytesUtils.utf8Bytes(requery);
        if (this.maxSize > size || !fileStore.supportNetwork()) {
            if (log.isWarnEnabled()) {
                log.warn("The summary store can not support network={} or the size is less than {}", this.store, size);
            }
            return null;
        }
        if (log.isInfoEnabled()) {
            log.info("The summary will be transferred to the media context.");
        }
        // 不能使用List.of，防止下游需要修改
        List<MediaContext> mediaContexts = new ArrayList<MediaContext>();
        MediaContext mediaContext = new MediaContext();
        mediaContext.setData(summaryConfig.getBase64() ? Base64.getEncoder().encodeToString(requery.getBytes(StandardCharsets.UTF_8)) : this.defStore.store(requery.getBytes(StandardCharsets.UTF_8), ".json", workTask));
        mediaContext.setType(this.buildMediaType(summaryConfig, workTask, histories));
        mediaContexts.add(mediaContext);
        return mediaContexts;
    }

    protected String buildReQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<MediaContext> mediaContext, String requery) throws Exception {
        return !CollectionUtils.isEmpty(mediaContext) ? this.requery : requery;
    }

    protected String buildQuery(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
        return JsonUtils.write(new LLMHistoriesPrompts(histories)) + (!StringUtils.isEmpty(append) ? System.lineSeparator() + append : append);
    }

    // 用于子类覆盖；若历史中无有效 created 则用任务时间戳兜底，避免 NPE 与误清
    protected Long buildLastTimeline(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        Long maxCreated = History.buildLastTimeline(histories);
        return maxCreated != null ? maxCreated : (workTask.getCreated() != null ? workTask.getCreated() : System.currentTimeMillis());
    }

    protected String buildMediaType(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        return summaryConfig.getBase64() ? MediaContext.PREFIX_INLINE + this.mimeType : this.mimeType;
    }

    protected SummaryPart buildSummaryPart(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
        Assert.hasText(content, "The summary content can not be empty");
        return SummaryPart.builder().pairs(summaryConfig.getSplit() ? this.buildPairs(summaryConfig, workTask, content) : null).content(content).build();
    }

    protected SummaryPart reset(SummaryConfig summaryConfig, WorkflowTask workTask, SummaryPart summaryPart, Long lastTime) throws Exception {
        List<String> repositories = summaryConfig.getRepositories(workTask.getWorkflow());
        // 清除并重新插入
        // 先 clear 再 store，避免摘要被 clear 误删；仅当解析出有效 pairs 时才落库，避免误清空历史
        Assert.notNull(this.historyStore, "The history store can not be empty");
        this.historyStore.clear(workTask, repositories, summaryConfig.getDesc(), -lastTime);
        this.historyStore.store(workTask, repositories, summaryPart.getPairs(), summaryConfig.getExpired(), summaryConfig.getMaxsize());
        return summaryPart;
    }

    protected SummaryPart update(SummaryConfig summaryConfig, WorkflowTask workTask, SummaryPart summaryPart, Long lastTime) throws Exception {
        // 更新LastTimeLine
        summaryPart.getPairs().forEach(pair -> {
            pair.setSource(pair.getSource() != null ? pair.getSource() : SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()));
            pair.setConversation(pair.getConversation() != null ? pair.getConversation() : workTask.getConversation());
            pair.setChat(pair.getChat() != null ? pair.getChat() : workTask.getChat());
            pair.setCreated(pair.getCreated() != null ? pair.getCreated() : lastTime);
        });
        return summaryPart;
    }

    // 是否进行条件判断
    protected Boolean allowed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories, String append) throws Exception {
        if (!summaryConfig.hasCondition()) {
            return true;
        }
        String reQuery = this.buildQuery(summaryConfig, workTask, histories, append);
        SyncConfig syncConfig = SyncConfig.builder().timeout(summaryConfig.getTimeout4Llm(this.timeout4Llm)).workflow(summaryConfig.getCondition()).workTask(workTask).reQuery(reQuery).build();
        String condition = StringUtils.lowerCase(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get());
        if (log.isDebugEnabled()) {
            log.debug("Summary condition={}-{}", reQuery, condition);
        }
        Assert.hasText(condition, "Summary condition can not be empty");
        // True: True/true/Yes/Y/1
        // False: False/false/No/N/0 and other
        // Json: {xxx,"condition":true/false/1/0}
        return ConditionUtils.checkCondition(condition).print().getCondition();
    }

    protected List<History> selectHistories(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        if (CollectionUtils.isEmpty(histories)) {
            return histories;
        }
        if (!summaryConfig.getIncludeFunCall()) {
            // 剔除Function类型的history，只保留非FunCall的会话
            return histories.stream()
                    .filter(h -> !h.isFunction(History.FUN_FUNCALL))
                    .collect(Collectors.toList());
        }
        return histories;
    }

    // 拆解Content，可以覆盖自定义解析
    protected List<HistoryPair> buildPairs(SummaryConfig summaryConfig, WorkflowTask workTask, String content) throws Exception {
        try {
            List<HistoryPair> pairs = List.of(JsonUtils.read(content, HistoryPair[].class));
            if (log.isDebugEnabled()) {
                log.debug("Summary content after using json={}", pairs);
            }
            return pairs.stream()
                    .peek(p -> {
                        p.setSource(SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()));
                        p.setConversation(workTask.getConversation());
                        // 使用默认
                        p.setApi(ProviderRequest.REQUEST_DEF);
                        p.setModel(ProviderRequestModel.DEF);
                        p.setChat(workTask.getChat());
                    })
                    .collect(Collectors.toList());
        } catch (JacksonException e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
        }
        String[] lines = content.split("=");
        if (log.isDebugEnabled()) {
            log.debug("Summary content after using text={}", Arrays.toString(lines));
        }
        List<HistoryPair> pairs = new ArrayList<>();
        content.lines().filter(line -> line.contains("=")).forEach(line -> {
            String[] parts = line.split("=");
            Assert.isTrue(parts.length == 2, "Summary content must contain only one key: " + line);
            Assert.hasText(parts[1], "Summary answer can not be empty");
            Assert.hasText(parts[0], "Summary query can not be empty");
            HistoryPair pair = new HistoryPair();
            pair.setSource(SplitUtils.join(workTask.getWorkflow(), workTask.getBiz()));
            pair.setConversation(workTask.getConversation());
            // 使用默认
            pair.setApi(ProviderRequest.REQUEST_DEF);
            pair.setModel(ProviderRequestModel.DEF);
            pair.setChat(workTask.getChat());
            pair.setAnswer(parts[1]);
            pair.setQuery(parts[0]);
            pairs.add(pair);
        });
        return pairs;
    }

    protected void dropOnFailed(SummaryConfig summaryConfig, WorkflowTask workTask, List<History> histories) throws Exception {
        try {
            if (summaryConfig.getDropOnFailed()) {
                Long lastTimeline = this.buildLastTimeline(summaryConfig, workTask, histories != null ? histories : workTask.getHistories());
                List<String> repositories = summaryConfig.getRepositories(workTask.getWorkflow());
                Assert.notNull(this.historyStore, "The history store can not be empty");
                this.historyStore.clear(workTask, repositories, summaryConfig.getDesc(), -lastTimeline);
                if (log.isWarnEnabled()) {
                    log.warn("The summary drop on failed, lastTimeline={}, repositories={}", lastTimeline, repositories);
                }
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    protected List<History> updateHistories(SummaryConfig summaryConfig, List<History> histories) throws Exception {
        histories.forEach(item -> {
            item.setReason(summaryConfig.getIncludeReason() ? item.getReason() : null);
            item.setSource(null);
        });
        return histories;
    }

    @Getter
    @JacksonXmlRootElement(localName = "Histories")
    public static class LLMHistoriesPrompts {

        @JacksonXmlElementWrapper(useWrapping = false)
        @JacksonXmlProperty(localName = "history")
        protected List<LLMHistoryPrompts> history;

        public LLMHistoriesPrompts(List<History> histories) {
            for (History history : histories) {
                this.add(new LLMHistoryPrompts(history));
            }
        }

        public LLMHistoriesPrompts add(LLMHistoryPrompts history) {
            if (this.history == null) {
                this.history = new ArrayList<LLMHistoryPrompts>();
            }
            this.history.add(history);
            return this;
        }
    }

    @Getter
    @JacksonXmlRootElement(localName = "history")
    public static class LLMHistoryPrompts {

        @JacksonXmlProperty(isAttribute = true)
        protected Integer deepness;

        @JacksonXmlProperty(isAttribute = true)
        protected String function;

        // 输入数据
        @JacksonXmlText
        protected Object content;

        @JacksonXmlProperty(isAttribute = true)
        protected Long created;

        @JacksonXmlProperty(isAttribute = true)
        protected String role;

        @JacksonXmlProperty(isAttribute = true)
        protected String type;

        public LLMHistoryPrompts(History history) {
            this.role = history.isRole(History.ROLE_ASSISTANT) ? "ASSISTANT" : "USER";
            this.type = history.isType(History.TYPE_ANSWER) ? "ANSWER" : "QUERY";
            this.function = history.getFunctionAsString();
            this.created = history.getCreated();
            this.content = history.getContent();
        }
    }

    @ConditionalOnProperty(name = "summary.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Autowired(required = false)
        protected HistoryStore historyStore;

        @Autowired
        protected DefStore defStore;

        @Value("${request.maxSize:10485760}")
        // LLM最大长度
        protected Integer provider4maxSize;

        @Value("${request.maxRate:0.5}")
        protected Double provider4maxRate;

        @Value("${summary.maxSize:}")
        protected Integer summary4maxSize;

        @Value("${summary.timeout.llm:1800000}")
        // 摘要调用下游思考链（Workflow）的超时
        protected Integer timeout4Llm;

        @Value("${summary.store.mimeType:text/plain}")
        protected String mimeType;

        @Value("${summary.requery:The content of the link needs to be summarized.}")
        protected String requery;

        @Value("${summary.store.name:}")
        protected String store;

        @Bean
        @ConditionalOnMissingBean(value = SummaryService.class)
        public SummaryService summaryService() throws Exception {
            SummaryServiceImpl summaryService = new SummaryServiceImpl();
            BeanUtils.copyProperties(this, summaryService);
            summaryService.setMaxSize(this.summary4maxSize != null ? this.summary4maxSize : (int) (this.provider4maxSize * this.provider4maxRate));
            log.info("SummaryServiceImpl inited, timeout4Llm={}", summaryService.getTimeout4Llm());
            return summaryService;
        }
    }
}
