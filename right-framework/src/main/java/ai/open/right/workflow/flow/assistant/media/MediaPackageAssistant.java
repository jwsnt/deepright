package ai.open.right.workflow.flow.assistant.media;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.media.MediaPackage;
import ai.open.right.workflow.flow.media.MediaPackageService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

@Slf4j
@Setter
@Getter
// 将Media URL和内容打包成Json（MediaPackage）
public class MediaPackageAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-mediaPackage";

    protected MediaPackageService mediaPackageService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        List<MediaPackage> content = this.mediaPackageService.pack(workflowConfig.getMediaConfig(), workTask);
        this.chainOr2Endpoint(workflowConfig, workTask, JsonUtils.write(content));
    }

    @ConditionalOnProperty(name = "media.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected MediaPackageService mediaPackageService;

        @Bean(MediaPackageAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = MediaPackageAssistant.WORKFLOW_NAME)
        public MediaPackageAssistant mediaPackageAssistant() throws Exception {
            MediaPackageAssistant mediaPackageAssistant = new MediaPackageAssistant();
            BeanUtils.copyProperties(this, mediaPackageAssistant);
            log.info("MediaPackageAssistant inited");
            return mediaPackageAssistant;
        }
    }
}
