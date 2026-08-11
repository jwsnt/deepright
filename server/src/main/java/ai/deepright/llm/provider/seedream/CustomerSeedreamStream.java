package ai.deepright.llm.provider.seedream;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamRequest;
import ai.open.right.workflow.flow.llm.provider.seedream.SeedreamStream;
import com.google.common.collect.ImmutableMap;

import java.io.File;
import java.nio.file.Paths;
import java.util.List;
import java.util.UUID;

public class CustomerSeedreamStream extends SeedreamStream {

    protected CliSubFetcher cliSubFetcher;

    public CustomerSeedreamStream(ProviderStreamConfig<SeedreamRequest> providerRequestConfig, CliSubFetcher cliSubFetcher) throws Exception {
        super(providerRequestConfig);
        this.cliSubFetcher = cliSubFetcher;
    }

    protected String download(String path) throws Exception {
        String file = this.buildFile(path);
        CliPubData pubData = this.cliSubFetcher.command(this.request.getMessage(), new RouterDevice(this.request.getMessage()), CliSubOps.builder()
                .app(List.of("curl", "mkdir"))
                .w(List.of(file))
                .exempted(true)
                .build(), CliPubSub.buildPushURL(this.request.getMessage(), path, file), "").valid();
        WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
        return file;
    }

    protected String buildSuffix(String path) throws Exception {
        return "png";
    }

    protected String buildFile(String path) throws Exception {
        return FeatureUtils.buildSysPath(this.request.getMessage(), Paths.get(FeatureUtils.buildWorkspace(this.request.getMessage())) + File.separator + "images" + File.separator + UUID.randomUUID() + "." + this.buildSuffix(path));
    }

    @Override
    protected String addUrlData(String path) throws Exception {
        return JsonUtils.write(ImmutableMap.of("file", this.download(path), "url", path));
    }
}
