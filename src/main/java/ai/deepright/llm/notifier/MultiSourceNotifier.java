package ai.deepright.llm.notifier;

import ai.deepright.feature.FeatureField;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.SourceNotifier;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

@Slf4j
public class MultiSourceNotifier extends SourceNotifier {

    public static final String MAIN = "main@main";

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        // Goal等价于Main处理
        super.notify(this.buildSegment(segment, redirectContext, notifierWriteBack, mediaContext), redirectContext, notifierWriteBack, mediaContext);
    }

    // 去掉Metadata
    protected Segment buildSegment(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        try {
            // 客户端气泡爆炸逻辑（workflow=main@main + workflow=__开头交替）
            Segment withOutMeta = segment.copyWithId();
            Map<String, Object> metadata = new HashMap<String, Object>();
            metadata.put(MultiSourceFlag.TASK_START, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.TASK_START));
            metadata.put(MultiSourceFlag.TASK_CLOSE, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.TASK_CLOSE));
            metadata.put(MultiSourceFlag.PROCESS, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.PROCESS));
            metadata.put(MultiSourceFlag.TARGET, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.TARGET));
            metadata.put(MultiSourceFlag.RESET, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.RESET));
            metadata.put(MultiSourceFlag.IMAGE, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.IMAGE));
            metadata.put(MultiSourceFlag.DELAY, MapUtils.getObject(withOutMeta.getMetadata(), MultiSourceFlag.DELAY));
            metadata.put(MultiSourceFlag.WARN, MapUtils.getInteger(withOutMeta.getMetadata(), MultiSourceFlag.WARN));
            metadata.put(MultiSourceFlag.TID, MapUtils.getString(withOutMeta.getMetadata(), MultiSourceFlag.TID));
            metadata.put(FeatureField.KEY_AGENTID, MapUtils.getString(segment.getMetadata(), FeatureField.KEY_AGENTID));
            metadata.values().removeIf(Objects::isNull);
            withOutMeta.putMetadata(metadata);
            return withOutMeta;
        } finally {
            // 标记消费
            this.rewriteSegment(segment.getContent(), redirectContext, notifierWriteBack, mediaContext);
            segment.mark();
        }
    }

    protected void rewriteSegment(String segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        // 用于后续多写通道（如审计日志）
        if (log.isDebugEnabled()) {
            log.debug("The multi rewrite content={}", segment);
        }
    }

    @Configuration
    public static class InitConfig {
        @Bean(SourceNotifier.SOURCE)
        @ConditionalOnMissingBean(name = SourceNotifier.SOURCE)
        public SourceNotifier sourceNotifier() throws Exception {
            MultiSourceNotifier sourceNotifier = new MultiSourceNotifier();
            log.info("MultiSourceNotifier inited");
            return sourceNotifier;
        }
    }
}
