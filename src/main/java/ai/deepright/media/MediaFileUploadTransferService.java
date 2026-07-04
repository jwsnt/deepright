package ai.deepright.media;

import ai.deepright.cli.CliPrinter;
import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.router.RouterDevice;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.impl.MediaTransferServiceImpl;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.List;

@Getter
@Setter
@Slf4j
public class MediaFileUploadTransferService extends MediaTransferServiceImpl {

    public static final String LANG_KEY_IMAGE_ERROR = "image.error";

    protected NotifierService notifierService;

    protected CliSubFetcher cliSubFetcher;

    protected SysStore sysStore;

    protected Integer timeout;

    @Override
    protected void initMediaContext(MediaConfig mediaConfig, WorkflowTask workTask, List<MediaContext> mediaContext, List<MediaResource> resources) throws Exception {
        // 初始化失败的列表
        for (int index = 0; index < resources.size(); index++) {
            try {
                resources.get(index).init();
            } catch (Exception e) {
                // 下载错误
                this.source(workTask, CliPrinter.brief(WorkflowException.code(e) + ": " + mediaContext.get(index).getData()));
                throw e;
            }
        }
    }

    // URI资源加载
    @Override
    protected MediaResource buildUriResource(MediaContext mediaContext, WorkflowTask workTask) throws Exception {
        workTask.markQuery();
        try {
            return MediaCliResource.builder()
                    .cliSubFetcher(this.cliSubFetcher)
                    .uri(mediaContext.getData())
                    .mediaContext(mediaContext)
                    .sysStore(this.sysStore)
                    .resource(this.resource)
                    .timeout(this.timeout)
                    .workTask(workTask)
                    .build();
        } finally {
            workTask.resetQuery();
        }
    }

    protected void source(WorkflowTask workTask, String content) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            // ![](http://127.0.0.1:9998/data?name=xxxx)
            StringBuffer buffer = new StringBuffer(System.lineSeparator());
            buffer.append(XmlResourceLang.get(MediaFileUploadTransferService.LANG_KEY_IMAGE_ERROR).replace("#content", content)).append(System.lineSeparator());
            this.notifierService.notify(Segment.build(workTask, Segment.SegmentConfig.builder()
                    .content(new StringBuffer(buffer.toString()))
                    .notifier(Notifier.SOURCE)
                    .build()), workTask);
        }
    }

    @Builder
    public static class MediaCliResource implements MediaResource {

        protected final CloseableHttpAsyncClient resource;

        protected final CliSubFetcher cliSubFetcher;

        protected final MediaContext mediaContext;

        protected final WorkflowTask workTask;

        protected final SysStore sysStore;

        protected final Integer timeout;

        protected final String uri;

        public void init() throws Exception {
            // 从客户端上传图片
            CliPubData pubData = this.cliSubFetcher.fetch(this.workTask, new RouterDevice(this.workTask), this.uri, "");
            Assert.isTrue(pubData.isOk(), pubData.getCmd());
            // Vertex容易触发400:URL_REJECTED-REJECTED_FC_TOO_MANY_PENDING，需要转成base64
            this.fullData(pubData);
        }

        protected void fullData(CliPubData pubData) throws Exception {
            if (pubData.isEncode(CliPubData.TEXT)) {
                // 纯文本（非二进制）
                this.mediaContext.setData(Base64.getEncoder().encodeToString(StringUtils.trim(StringUtils.defaultIfEmpty(StringUtils.substringAfter(pubData.getCmd(), StringUtils.LF), pubData.getCmd())).getBytes(StandardCharsets.UTF_8)));
            } else {
                // 强制转换为文本（内部编码base64）
                this.mediaContext.setData(pubData.forceText(this.resource, this.sysStore, this.timeout, true).getCmd());
            }
            // 错误模型需要使用
            Assert.hasText(this.mediaContext.getData(), pubData.getCmd());
            // 重写Mime Type
            this.mediaContext.setType(MediaContext.PREFIX_INLINE + this.mediaContext.getType());
        }
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Getter
    @Setter
    @Configuration
    public static class CliInitConfig extends InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Autowired
        protected SysStore sysStore;

        // 毫秒（1分钟）
        @Value("${media.transfer.timeout:60000}")
        protected Integer timeout;

        @Override
        @Bean
        @ConditionalOnMissingBean(value = MediaTransferServiceImpl.class)
        public MediaFileUploadTransferService mediaTransferService() throws Exception {
            MediaFileUploadTransferService mediaTransferService = new MediaFileUploadTransferService();
            BeanUtils.copyProperties(this, mediaTransferService);
            log.info("MediaFileUploadTransferService inited");
            return mediaTransferService;
        }
    }
}
