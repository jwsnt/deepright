package ai.open.right.workflow.flow.media.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.media.MediaInlineData;
import ai.open.right.workflow.flow.media.MediaInlineService;
import com.google.common.net.MediaType;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;

@Getter
@Setter
@Slf4j
public class MediaInlineServiceImpl implements MediaInlineService {

    private final Map<MediaType, String> suffix = new HashMap<>();

    protected FileStore fileStore;

    @PostConstruct
    public void init() throws Exception {
        this.buildSuffix();
    }

    @Override
    public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
        Assert.notNull(mediaInlineData.getMediaType(), "Media media type can not be empty");
        Assert.notNull(mediaInlineData.getData(), "Media inline data can not be empty");
        String resource = this.fileStore.store(Base64.getDecoder().decode(mediaInlineData.getData().getBytes(StandardCharsets.UTF_8)), this.buildSuffix(mediaInlineData, workTask), workTask);
        return this.buildResource(workTask, resource);
    }

    protected String buildSuffix(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
        MediaType parsed = MediaType.parse(mediaInlineData.getMediaType().trim());
        return this.suffix.get(parsed.withoutParameters());
    }

    // 用于子类覆盖
    protected String buildResource(WorkflowTask workTask, String resource) throws Exception {
        return resource;
    }

    protected void buildSuffix() throws Exception {
        this.suffix.put(MediaType.create("application", "json"), ".json");
        this.suffix.put(MediaType.create("application", "pdf"), ".pdf");
        this.suffix.put(MediaType.create("application", "xml"), ".xml");
        this.suffix.put(MediaType.create("text", "plain"), ".txt");
        this.suffix.put(MediaType.create("text", "html"), ".html");
        this.suffix.put(MediaType.create("image", "png"), ".png");
        this.suffix.put(MediaType.create("image", "jpeg"), ".jpg");
        this.suffix.put(MediaType.create("image", "gif"), ".gif");
    }

    @ConditionalOnProperty(name = "media.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(DefStore.NAME)
        protected FileStore fileStore;

        @Bean
        @ConditionalOnMissingBean(value = MediaInlineService.class)
        public MediaInlineService mediaInlineService() throws Exception {
            MediaInlineServiceImpl mediaInlineService = new MediaInlineServiceImpl();
            BeanUtils.copyProperties(this, mediaInlineService);
            log.info("MediaTransferServiceImpl inited");
            return mediaInlineService;
        }
    }
}
