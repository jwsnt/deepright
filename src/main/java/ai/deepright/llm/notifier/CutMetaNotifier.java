package ai.deepright.llm.notifier;

import ai.open.right.context.RedirectContext;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.EndpointNotifier;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.util.List;

@Slf4j
public class CutMetaNotifier extends EndpointNotifier {

    public static final String WORKFLOW = "cli@get";

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        super.notify(this.buildSegment(segment, redirectContext, notifierWriteBack, mediaContext), redirectContext, notifierWriteBack, mediaContext);
    }

    protected Segment buildSegment(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        if (StringUtils.equalsIgnoreCase(CutMetaNotifier.WORKFLOW, SplitUtils.join(segment))) {
            segment = segment.copyWithId();
            segment.delMetadata();
        }
        return segment;
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    public static class InitConfig {

        @Bean(Notifier.ENDPOINT)
        @ConditionalOnMissingBean(name = Notifier.ENDPOINT)
        public CutMetaNotifier endpointNotifier() throws Exception {
            CutMetaNotifier endpointNotifier = new CutMetaNotifier();
            log.info("CutMetaNotifier inited");
            return endpointNotifier;
        }
    }
}
