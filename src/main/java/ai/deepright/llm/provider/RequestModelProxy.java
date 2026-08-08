package ai.deepright.llm.provider;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.nio.file.Paths;
import java.util.List;
import java.util.Map;

public class RequestModelProxy {

    public static void configProxy(CliSubFetcher cliSubFetcher, RequestProxyConfig proxyConfig) throws Exception {
        String app = FeatureUtils.buildApp(proxyConfig.getMetadata());
        String command = FeatureUtils.escapeShell(FeatureFlag.isWindows(proxyConfig.getMetadata()), app) + " token --provider " + FeatureUtils.escapeShell(FeatureFlag.isWindows(proxyConfig.getMetadata()), proxyConfig.getProvider());
        CliSubOps subOps = CliSubOps.builder().app(List.of(Paths.get(app).getFileName().toString())).r(List.of(command)).exempted(true).build();
        CliPubData pubData = proxyConfig.isRouter() ? cliSubFetcher.command(proxyConfig.getWorkTask(), proxyConfig.getRouterDevice(), subOps, command, "") : cliSubFetcher.command(proxyConfig.getWorkTask(), subOps, command, "");
        WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
        Map<String, String> provider = MapUtils.getMap(JsonUtils.read(pubData.getCmd(), Map.class), proxyConfig.getProvider());
        String multiOutput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_OUTPUT);
        String multiInput = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_MULTI_INPUT);
        String thinking = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_THINKING);
        String base = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_BASE);
        String fast = MapUtils.getString(provider, RequestModelSelect.KEY_MODEL_FAST);
        String token = MapUtils.getString(provider, "token");
        String url = MapUtils.getString(provider, "__url");
        WorkflowException.checkCondition(StringUtils.isEmpty(token), "The proxy provider token can not be empty: " + proxyConfig.getProvider());
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_MULTI_OUTPUT, !StringUtils.isEmpty(multiOutput) ? multiOutput : null);
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_MULTI_INPUT, !StringUtils.isEmpty(multiInput) ? multiInput : null);
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_THINKING, !StringUtils.isEmpty(thinking) ? thinking : null);
        proxyConfig.getMetadata().put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_BASE, !StringUtils.isEmpty(base) ? base : null);
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_FAST, !StringUtils.isEmpty(fast) ? fast : null);
        proxyConfig.getMetadata().put(RequestModelSelect.KEY_MODEL_URL, !StringUtils.isEmpty(url) ? url : null);
        proxyConfig.getMetadata().put(ProviderRequestService.KEY_PROVIDER, proxyConfig.getProvider());
    }

    // 提取当前Loop的代理模型
    public static String buildProxy(WorkflowTask workTask) throws Exception {
        String proxy = null;
        proxy = RequestModelSelect.multiOutput(workTask) ? RequestModelSelect.proxyMultiOutput(workTask) : proxy;
        proxy = RequestModelSelect.multiInput(workTask) ? RequestModelSelect.proxyMultiInput(workTask) : proxy;
        return proxy;
    }

    @Builder
    @Getter
    @Setter
    public static class RequestProxyConfig {

        protected Map<String, Object> metadata;

        protected RouterDevice routerDevice;

        protected WorkflowTask workTask;

        protected String provider;

        public Boolean isRouter() throws Exception {
            return this.routerDevice != null;
        }
    }
}
