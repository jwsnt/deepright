package ai.deepright.workflow;

import ai.deepright.auth.AuthService;
import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.llm.notifier.MultiSourceNotifier;
import ai.deepright.llm.provider.RequestModelSelect;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.Workflow;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.impl.WorkflowImpl;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.nio.file.Paths;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class ReConfigWorkflow extends WorkflowImpl {

    public static final String NAME = "workflow";

    protected CliSubFetcher cliSubFetcher;

    protected AuthService authService;

    @Override
    protected WorkflowTask reInitial(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 原有逻辑
        workTask = super.reInitial(workflowConfig, workTask);
        this.proxyMulti(workTask);
        this.provider(workTask);
        this.block(workTask);
        return workTask;
    }

    // 代理模型
    protected void proxyMulti(WorkflowTask workTask) throws Exception {
        if (RequestModelSelect.isProxyAvailable(workTask)) {
            String proxy = this.buildProxy(workTask);
            String app = FeatureUtils.buildApp(workTask);
            String path = FeatureUtils.escapePath(FeatureFlag.isWindows(workTask), app + " token --provider " + proxy);
            CliPubData pubData = this.cliSubFetcher.command(workTask, CliSubOps.builder()
                    .app(List.of(Paths.get(app).getFileName().toString()))
                    .r(List.of(path))
                    .exempted(true)
                    .build(), path, "");
            WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
            Map<String, String> provider = MapUtils.getMap(JsonUtils.read(pubData.getCmd(), Map.class), proxy);
            String multiOutput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_OUTPUT);
            String multiInput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_INPUT);
            String token = MapUtils.getString(provider, "token");
            String url = MapUtils.getString(provider, "__url");
            WorkflowException.checkCondition(StringUtils.isEmpty(token), "The proxy provider token can not be empty: " + proxy);
            workTask.putMetadata(RequestModelSelect.KEY_MODEL_MULTI_OUTPUT, !StringUtils.isEmpty(multiOutput) ? multiOutput : null);
            workTask.putMetadata(RequestModelSelect.KEY_MODEL_MULTI_INPUT, !StringUtils.isEmpty(multiInput) ? multiInput : null);
            workTask.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
            workTask.putMetadata(RequestModelSelect.KEY_MODEL_URL, !StringUtils.isEmpty(url) ? url : null);
            workTask.putMetadata(ProviderRequestService.KEY_PROVIDER, proxy);
        }
    }

    protected String buildProxy(WorkflowTask workTask) throws Exception {
        String proxy = null;
        proxy = RequestModelSelect.multiOutput(workTask) ? RequestModelSelect.proxyMultiOutput(workTask) : proxy;
        proxy = RequestModelSelect.multiInput(workTask) ? RequestModelSelect.proxyMultiInput(workTask) : proxy;
        return proxy;
    }

    @Override
    protected void rateLimit(WorkflowTask workTask) throws Exception {
        if (StringUtils.equalsIgnoreCase(MultiSourceNotifier.MAIN, SplitUtils.join(workTask))) {
            super.rateLimit(workTask);
        }
    }

    protected void provider(WorkflowTask workTask) throws Exception {
        String provider = FeatureUtils.buildTargetProvider(workTask);
        if (this.authService.support(provider)) {
            this.authService.auth(workTask, provider, MapUtils.getString(workTask.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN));
        }
    }

    protected void block(WorkflowTask workTask) throws Exception {
        if (workTask.isEntry()) {
            // @See CliSubBlocker 终止之前同设备同会话并行任务
            this.blockService.submit("main", workTask.getChat(), RouterDevice.key(workTask), workTask, workTask.getCreated() - 1);
        }
    }

    @Configuration
    @Getter
    @Setter
    public static class ReConfigInitConfig extends InitConfig {

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected AuthService authService;

        @Override
        @Bean(ReConfigWorkflow.NAME)
        @ConditionalOnMissingBean(value = Workflow.class)
        public Workflow workflow() throws Exception {
            ReConfigWorkflow workflow = new ReConfigWorkflow();
            BeanUtils.copyProperties(this, workflow);
            log.info("ReConfigWorkflow inited: deepness={}, messageOnFailed={}", workflow.getDeepness(), workflow.getMessageOnFailed());
            return workflow;
        }
    }
}
