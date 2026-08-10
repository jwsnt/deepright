package ai.open.right.workflow.flow.media.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.media.MediaConfig;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaPackage;
import ai.open.right.workflow.flow.media.MediaPackageService;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
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

@Setter
@Getter
@Slf4j
public class MediaPackageServiceImpl implements MediaPackageService {

    protected NotifierService notifierService;

    // Media Package调用下游思考链（Workflow）超时
    protected Integer timeout4Llm;

    // Media Package切分符
    private String split = System.lineSeparator();

    @Override
    public List<MediaPackage> pack(MediaConfig mediaConfig, WorkflowTask workTask) throws Exception {
        SyncConfig syncConfig = SyncConfig.builder()
                // 指定Workflow，默认使用KEY_FUN_MEDIA
                .workflow(mediaConfig != null ? mediaConfig.getDynamic(ProviderRequestService.KEY_FUN_MEDIA) : ProviderRequestService.KEY_FUN_MEDIA)
                .timeout(mediaConfig != null ? mediaConfig.getTimeout4Llm(this.timeout4Llm) : this.timeout4Llm)
                .mediaContext(workTask.getMediaContext())
                .workTask(workTask)
                .build();
        String response = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
        if (log.isDebugEnabled()) {
            log.debug("Media package response={}", response);
        }
        Assert.hasText(response, "Media package response can not be empty");
        String[] content = response.split(mediaConfig != null ? mediaConfig.getSplit(this.split) : this.split);
        if (log.isInfoEnabled()) {
            log.info("Media package response after splitting={}", Arrays.toString(content));
        }
        // 切分数量检查
        Assert.isTrue(workTask.getMediaContext().size() == content.length, "Media response counts are unequal: " + workTask.getMediaContext().size() + "/" + content.length);
        List<MediaPackage> mediaPackages = new ArrayList<MediaPackage>();
        for (int index = 0; index < workTask.getMediaContext().size(); index++) {
            MediaContext mediaContext = workTask.getMediaContext().get(index);
            mediaPackages.add(MediaPackage.builder().source(mediaContext.getData()).content(content[index]).build());
        }
        return mediaPackages;
    }

    @ConditionalOnProperty(name = "media.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected NotifierService notifierService;

        @Value("${media.package.timeout.llm:1800000}")
        // Media Package调用下游思考链（Workflow）超时
        protected Integer timeout4Llm;

        @Value("${media.package.split:\n}")
        // Media Package切分符
        private String split = System.lineSeparator();

        @Bean
        @ConditionalOnMissingBean(value = MediaPackageService.class)
        public MediaPackageService mediaPackageService() throws Exception {
            MediaPackageServiceImpl mediaPackageService = new MediaPackageServiceImpl();
            BeanUtils.copyProperties(this, mediaPackageService);
            log.info("MediaPackageServiceImpl inited, timeout4Llm={},split={}", mediaPackageService.getTimeout4Llm(), mediaPackageService.getSplit());
            return mediaPackageService;
        }
    }
}
