package ai.open.right.workflow.flow.assistant.media;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.media.MediaContent;
import ai.open.right.workflow.flow.media.MediaTransferService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

@Slf4j
@Setter
@Getter
// 将Media URL转换为Base64
public class MediaTransferAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-mediaTransfer";

    protected MediaTransferService mediaTransferService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        MediaContent mediaContent = workTask.getObjectQuery(MediaContent.class);
        // Query检查
        Assert.isTrue(mediaContent != null && mediaContent.hasQuery(), "Media query can not be empty, please check if the JSON is compatible with the `MediaContent` format: " + workTask.getQuery());
        this.buildMediaContext(workflowConfig, workTask, mediaContent);
        this.buildMetadata(workTask, mediaContent);
        this.chainOr2Endpoint(workflowConfig, workTask, mediaContent.getMediaContext(), mediaContent.getQuery());
    }

    protected void buildMediaContext(WorkflowConfig workflowConfig, WorkflowTask workTask, MediaContent mediaContent) throws Exception {
        if (mediaContent.hasMediaContext()) {
            // 需要进行转换
            this.transfer(workflowConfig, workTask, mediaContent);
        }
    }

    protected void buildMetadata(WorkflowTask workTask, MediaContent mediaContent) throws Exception {
        if (mediaContent.hasMetadata()) {
            for (String key : mediaContent.getMetadata().keySet()) {
                workTask.putMetadata(key, mediaContent.getMetadata().get(key));
            }
        }
    }

    protected void transfer(WorkflowConfig workflowConfig, WorkflowTask workTask, MediaContent mediaContent) throws Exception {
        if (workflowConfig.hasMedia()) {
            this.mediaTransferService.transfer(workflowConfig.getMediaConfig(), workTask, mediaContent.getMediaContext());
        }
    }

    @ConditionalOnProperty(name = "media.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected MediaTransferService mediaTransferService;

        @Bean(MediaTransferAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = MediaTransferAssistant.WORKFLOW_NAME)
        public MediaTransferAssistant mediaTransferAssistant() throws Exception {
            MediaTransferAssistant mediaTransferAssistant = new MediaTransferAssistant();
            BeanUtils.copyProperties(this, mediaTransferAssistant);
            log.info("MediaTransferAssistant inited");
            return mediaTransferAssistant;
        }
    }
}
