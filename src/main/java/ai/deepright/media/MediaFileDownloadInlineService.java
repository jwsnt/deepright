package ai.deepright.media;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliPubSub;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.module.HttpProtocol;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import ai.open.right.workflow.flow.media.impl.MediaInlineServiceImpl;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.io.FilenameUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.util.Assert;

import java.io.File;
import java.net.URI;
import java.nio.file.Paths;
import java.util.List;
import java.util.UUID;

@Slf4j
@Setter
@Getter
public class MediaFileDownloadInlineService extends MediaInlineServiceImpl {

    protected CliSubFetcher cliSubFetcher;

    protected HttpProtocol httpProtocol;

    @Override
    protected String buildResource(WorkflowTask workTask, String resource) throws Exception {
        try {
            // 将图片URL推送回端
            String http = this.buildUrl(workTask, resource);
            String file = this.download(workTask, http, this.buildFile(workTask, resource));
            // 如果推送成功就使用2个地址
            return JsonUtils.write(ImmutableMap.of("url", http, "file", file));
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return resource;
        }
    }

    protected String download(WorkflowTask workTask, String resource, String file) throws Exception {
        CliPubData pubData = this.cliSubFetcher.command(workTask, new RouterDevice(workTask), CliSubOps.builder()
                .app(List.of("curl", "mkdir"))
                .w(List.of(file))
                .exempted(true)
                .build(), CliPubSub.buildPushURL(workTask, resource, file), "").valid();
        Assert.isTrue(pubData.isOk(), pubData.getCmd());
        return file;
    }

    protected String buildSuffix(WorkflowTask workTask, String resource) throws Exception {
        return FilenameUtils.getExtension(new URI(resource).getPath());
    }

    protected String buildFile(WorkflowTask workTask, String resource) throws Exception {
        return FeatureUtils.buildSysPath(workTask, Paths.get(FeatureUtils.buildWorkspace(workTask)) + File.separator + "images" + File.separator + UUID.randomUUID() + "." + this.buildSuffix(workTask, resource));
    }

    protected String buildUrl(WorkflowTask workTask, String resource) throws Exception {
        return !MediaTransferUtils.isNetwork(resource) ? this.httpProtocol.dataHost(resource) : resource;
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Getter
    @Setter
    public static class CliInitConfig extends InitConfig {

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected HttpProtocol httpProtocol;

        @Override
        @Bean
        @ConditionalOnMissingBean(value = MediaInlineService.class)
        public MediaFileDownloadInlineService mediaInlineService() throws Exception {
            MediaFileDownloadInlineService mediaInlineService = new MediaFileDownloadInlineService();
            BeanUtils.copyProperties(this, mediaInlineService);
            log.info("MediaFileDownloadInlineService inited");
            return mediaInlineService;
        }
    }
}
