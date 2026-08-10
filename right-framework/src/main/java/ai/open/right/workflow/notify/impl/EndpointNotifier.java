package ai.open.right.workflow.notify.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

@Slf4j
// 推送至上一层思考链（Workflow）
public class EndpointNotifier implements Notifier {

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        if (!segment.getSilent()) {
            notifierWriteBack.writeBack(segment);
        }
    }

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, redirectContext, notifierWriteBack, null);
    }

    @Override
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, mediaContext);
    }

    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, null);
    }
    @Configuration
    public static class InitConfig {

        @Bean(Notifier.ENDPOINT)
        @ConditionalOnMissingBean(name = Notifier.ENDPOINT)
        public EndpointNotifier endpointNotifier() throws Exception {
            EndpointNotifier endpointNotifier = new EndpointNotifier();
            log.info("EndpointNotifier inited");
            return endpointNotifier;
        }
    }
}
