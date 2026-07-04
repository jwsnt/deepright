package ai.deepright.workflow.assistant;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.media.MediaTransferAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.notify.Notifier;
import ai.deepright.cli.CliPrinter;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.lang.XmlResourceLang;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
// 用于提醒资源生成时等待
public class ImageRemindAssistant extends MediaTransferAssistant {

    public static final String LANG_KEY_IMAGE_REMIND = "image.remind";

    @Override
    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        this.notify(workflowConfig, workTask);
        super.execute(workflowConfig, workTask);
    }

    protected void notify(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        if (!FeatureFlag.isSilent(workTask)) {
            this.notify(workTask, CliPrinter.process(ImageRemindAssistant.LANG_KEY_IMAGE_REMIND), Notifier.SOURCE, XmlResourceLang.get(ImageRemindAssistant.LANG_KEY_IMAGE_REMIND));
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class DefaultInitConfig extends InitConfig {

        @Bean(MediaTransferAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = MediaTransferAssistant.WORKFLOW_NAME)
        public ImageRemindAssistant mediaTransferAssistant() throws Exception {
            ImageRemindAssistant remindMediaAssistant = new ImageRemindAssistant();
            BeanUtils.copyProperties(this, remindMediaAssistant);
            log.info("RemindMediaAssistant inited");
            return remindMediaAssistant;
        }
    }
}
