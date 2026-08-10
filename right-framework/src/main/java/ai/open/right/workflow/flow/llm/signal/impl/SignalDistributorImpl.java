package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.signal.SignalConfig;
import ai.open.right.workflow.flow.llm.signal.SignalDistributor;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

@Slf4j
@Setter
@Getter
public class SignalDistributorImpl implements SignalDistributor {

    public static final String SPLIT_KV = "=";

    protected NotifierService notifierService;

    // Signal调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    public void distribute(SignalConfig signalConfig, String signal, Message message) throws Exception {
        String[] original = signal.split(SignalDistributorImpl.SPLIT_KV);
        if (log.isDebugEnabled()) {
            log.debug("The original signal={}", Arrays.toString(original));
        }
        // length=1，use original query
        // length，use original[1]
        // other，original[1]-N
        String response = original.length == 1 ? message.getQuery() : original.length == 2 ? original[1] : StringUtils.join(Arrays.copyOfRange(original, 1, original.length), "=");
        String workflow = signalConfig.getDynamic(original[0]);
        if (log.isInfoEnabled()) {
            log.info("The signal response and workflow: response={},workflow={}", response, workflow);
        }
        this.notify(message, response, workflow);
    }

    public void distribute(SignalConfig signalConfig, List<String> signal, Message message) throws Exception {
        List<SyncWorkflowTask> syncWorkflowTasks = this.getSyncWorkflowTasks(signalConfig, signal, message);
        String response = this.getSignalResponse(syncWorkflowTasks);
        this.notify(message, response, signalConfig.getSynthesizer());
    }

    protected List<SyncWorkflowTask> getSyncWorkflowTasks(SignalConfig signalConfig, List<String> signal, Message message) throws Exception {
        List<SyncWorkflowTask> syncWorkflowTasks = new ArrayList<SyncWorkflowTask>();
        for (String each : signal) {
            String[] original = each.split(SignalDistributorImpl.SPLIT_KV);
            if (log.isDebugEnabled()) {
                log.debug("The original signal={}", Arrays.toString(original));
            }
            // length=1，use original query
            // length，use pair[1]
            // other，pair[1]-N
            String workflow = signalConfig.getDynamic(original[0]);
            if (!StringUtils.isEmpty(workflow)) {
                String content = original.length == 1 ? message.getQuery() : original.length == 2 ? original[1] : Arrays.toString(Arrays.copyOfRange(original, 1, original.length - 1));
                Assert.hasText(content, "Signal content can not be empty");
                SyncConfig syncConfig = SyncConfig.builder()
                        .timeout(signalConfig.getTimeout4Llm(this.timeout4Llm))
                        .workflow(workflow)
                        .workTask(message)
                        .reQuery(content)
                        .build();
                syncWorkflowTasks.add(SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig));
            }
        }
        return syncWorkflowTasks;
    }

    protected String getSignalResponse(List<SyncWorkflowTask> syncWorkflowTasks) throws Exception {
        StringBuffer signalResponse = new StringBuffer();
        for (SyncWorkflowTask syncWorkflowTask : syncWorkflowTasks) {
            String body = syncWorkflowTask.get();
            if (!StringUtils.isEmpty(body)) {
                signalResponse.append(body);
            }
        }
        return signalResponse.toString();
    }

    protected void notify(Message message, String response, String workflow) throws Exception {
        if (log.isInfoEnabled()) {
            log.info("Signal notify={}-{}", workflow, response);
        }
        Assert.hasText(response, "Signal response can not be empty");
        Assert.hasText(workflow, "Signal workflow can not be empty");
        Segment.SegmentConfig config = Segment.SegmentConfig.builder()
                .content(new StringBuffer(response))
                .notifier(Notifier.LOCALHOST)
                .workflow(workflow)
                .build();
        // Not support customer notifier
        this.notifierService.notify(Segment.build(message, config), message, message);
    }

    @ConditionalOnProperty(name = "signal.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${signal.timeout.llm:1800000}")
        // Signal调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Bean
        @ConditionalOnMissingBean(value = SignalDistributor.class)
        public SignalDistributor signalDistributor() throws Exception {
            SignalDistributorImpl signalDistributor = new SignalDistributorImpl();
            BeanUtils.copyProperties(this, signalDistributor);
            log.info("SignalDistributorImpl inited: timeout4Llm={}", signalDistributor.getTimeout4Llm());
            return signalDistributor;
        }
    }
}
