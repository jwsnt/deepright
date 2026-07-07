package ai.deepright.cli.insert;

import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.notifier.MultiSourceFlag;
import ai.deepright.llm.provider.RequestContextBuilder;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.notify.Notifier;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

@Getter
@Setter
@Slf4j
public class CliInsertRag extends RagCondition implements CliInsertService, RagService, CliRecall {

    public static final String KEY_RECALL = "cli.recall";

    public static final String KEY_INSERT = "cli.insert";

    public static final String RAG_KEY = "rag_insert";

    protected ResourceService resourceService;

    protected HistoryStore historyStore;

    protected String template4insert;

    protected Integer maxSize;

    @PostConstruct
    public void init() throws Exception {
        this.template4insert = IOUtils.toString(new BufferedInputStream(this.resourceService.url(this.template4insert).openStream()), StandardCharsets.UTF_8);
        Assert.hasText(this.template4insert, "The template insert must not be empty");
    }

    @Override
    public List<History> recall(WorkflowTask workTask) throws Exception {
        return List.class.cast(MapUtils.getObject(workTask.getUserContext().getMetadata(), CliInsertRag.KEY_RECALL));
    }

    @Override
    public void insert(WorkflowTask workTask, List<CliInsert> inserts) throws Exception {
        // 放在UserContext(CurrentMap)，防止跨子任务丢失
        List<CliInsert> current = List.class.cast(MapUtils.getObject(workTask.getUserContext().getMetadata(), CliInsertRag.KEY_INSERT));
        (current = current != null ? current : new ArrayList<CliInsert>()).addAll(inserts);
        workTask.getUserContext().putMetadata(CliInsertRag.KEY_INSERT, current);
        this.notify(workTask);
        if (log.isInfoEnabled()) {
            log.info("The message was inserted={}", current.size());
        }
    }

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        this.storeQuery(ragConfig, ragData);
        this.recallQuery(ragConfig, ragData);
        return new RagAtOnce(ragConfig);
    }

    protected HistoryPair buildHistoryPair(RagConfig ragConfig, RagData ragData, CliInsert insert) throws Exception {
        // 当前时间
        HistoryPair history = new HistoryPair(ragData.getRequest().getMessage(), System.currentTimeMillis());
        history.setModel(ragData.getRequest().getModel());
        history.setApi(ragData.getRequest().getApi());
        history.setQuery(insert.getMessage());
        history.setRole(History.ROLE_USER);
        return history;
    }

    protected void storeHistory(RagConfig ragConfig, RagData ragData, List<HistoryPair> histories) throws Exception {
        if (ragData.getRequest().getContainHistories()) {
            this.historyStore.store(ragData.getRequest().getMessage(), ragData.getRequest().getRepositories(), histories, ragData.getRequest().getExpired(), ragData.getRequest().getHistories());
        }
    }

    protected void storeRecall(RagConfig ragConfig, RagData ragData, List<History> recall) throws Exception {
        if (recall.size() > this.maxSize) {
            recall.subList(0, recall.size() - this.maxSize).clear();
        }
        ragData.getRequest().getMessage().getUserContext().putMetadata(CliInsertRag.KEY_RECALL, recall);
    }

    protected void storeQuery(RagConfig ragConfig, RagData ragData) throws Exception {
        // 获取并删除
        List<CliInsert> inserts = ragData.getRequest().getMessage().getUserContext().delMetadata(CliInsertRag.KEY_INSERT, List.class);
        if (!CollectionUtils.isEmpty(inserts)) {
            List<History> recall = List.class.cast(MapUtils.getObject(ragData.getRequest().getMessage().getUserContext().getMetadata(), CliInsertRag.KEY_RECALL));
            // 复制，防止使用时写入
            recall = recall != null ? new ArrayList<History>(recall) : new ArrayList<History>();
            List<HistoryPair> histories = new ArrayList<HistoryPair>();
            for (CliInsert insert : inserts) {
                HistoryPair history = this.buildHistoryPair(ragConfig, ragData, insert);
                recall.add(history.buildHistories()[0]);
                histories.add(history);
            }
            this.storeHistory(ragConfig, ragData, histories);
            this.storeRecall(ragConfig, ragData, recall);
            this.notify(ragData.getQuery(), inserts);
            if (log.isInfoEnabled()) {
                log.info("The message was recalled={}", recall.size());
            }
        }
    }

    protected void recallQuery(RagConfig ragConfig, RagData ragData) throws Exception {
        List<History> recall = this.recall(ragData.getQuery());
        if (!CollectionUtils.isEmpty(recall)) {
            // 保留时间晚于Query.created的
            recall = recall.stream()
                    .filter(h -> h.getCreated() > ragData.getQuery().getCreated())
                    .collect(Collectors.toList());
            if (!CollectionUtils.isEmpty(recall)) {
                // 最后一条强化Query（会破坏最后一轮的KV缓存）
                ragData.getQuery().getHistories().addAll(recall);
                ragData.getQuery().getHistories().add(RequestContextBuilder.buildContext(ragData.getRequest(), this.template4insert, recall.getLast().getCreated() + RequestContextBuilder.NEXT));
                if (log.isInfoEnabled()) {
                    log.info("The recall histories={}", recall.size());
                }
            }
        }
    }

    @Override
    protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
        // 不为Task和后台任务
        return super.allowed(ragConfig, ragData) && !FeatureFlag.isTask(ragData.getQuery()) && !FeatureFlag.isDaemon(ragData.getQuery()) && !FeatureFlag.isSilent(ragData.getQuery());
    }

    protected void notify(WorkflowTask workTask, List<CliInsert> inserts) throws Exception {
        for (CliInsert insert : inserts) {
            Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                    .metadata(CliPrinter.process(CliInsertRag.RAG_KEY, MultiSourceFlag.TID, insert.getTid()))
                    .content(new StringBuffer(XmlResourceLang.get(CliInsertRag.KEY_RECALL)))
                    .workflow(workTask.getWorkflow())
                    .notifier(Notifier.SOURCE)
                    .build();
            this.notifierService.notify(Segment.build(workTask, segmentConfig), workTask, workTask);
        }
    }

    protected void notify(WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            Segment.SegmentConfig segmentConfig = Segment.SegmentConfig.builder()
                    .content(new StringBuffer(XmlResourceLang.get(CliInsertRag.KEY_INSERT)))
                    .metadata(CliPrinter.process(CliInsertRag.RAG_KEY))
                    .workflow(workTask.getWorkflow())
                    .notifier(Notifier.SOURCE)
                    .build();
            this.notifierService.notify(Segment.build(workTask, segmentConfig), workTask, workTask);
        }
    }


    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired
        protected ResourceService resourceService;

        @Autowired
        protected HistoryStore historyStore;

        @Value("${cli.insert.query:classpath:config/cli/insert.md}")
        protected String template4insert;

        @Value("${cli.insert.maxSize:15}")
        protected Integer maxSize;

        @Bean(CliInsertRag.RAG_KEY)
        @ConditionalOnMissingBean(name = CliInsertRag.RAG_KEY)
        public CliInsertRag cliInsertRag() throws Exception {
            CliInsertRag cliInsertRag = new CliInsertRag();
            BeanUtils.copyProperties(this, cliInsertRag);
            log.info("CliInsertRag inited, timeout4Condition={}", cliInsertRag.getTimeout4Condition());
            return cliInsertRag;
        }
    }
}
