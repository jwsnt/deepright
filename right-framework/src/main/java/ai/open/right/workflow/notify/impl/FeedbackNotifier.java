package ai.open.right.workflow.notify.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
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

import java.util.List;

@Slf4j
@Setter
@Getter
// 思考链（Localhost）和终端（Source）多路推送
public class FeedbackNotifier implements Notifier {

    protected Notifier localhostNotifier;

    protected Notifier endpointNotifier;

    protected Notifier sourceNotifier;

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        if (!segment.isFromFunMerge()) {
            // FunCall Merge会在父线程推送
            this.getSourceNotifier().notify(segment, redirectContext, notifierWriteBack, mediaContext);
        }
        if (segment.isFinished()) {
            if (!segment.isFromFunCall()) {
                this.getLocalhostNotifier().notify(segment.copyWithStart(0), redirectContext, notifierWriteBack, mediaContext);
            } else {
                // FunCall请求强制回流
                this.getEndpointNotifier().notify(segment.copyWithStart(0), redirectContext, notifierWriteBack, mediaContext);
            }
        }
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, null);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, mediaContext);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, null);
    }

    protected Notifier getLocalhostNotifier() {
        return this.localhostNotifier;
    }

    protected Notifier getSourceNotifier() {
        return this.sourceNotifier;
    }

    @ConditionalOnProperty(name = "feedback.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier(Notifier.LOCALHOST)
        protected Notifier localhostNotifier;

        @Autowired
        @Qualifier(Notifier.ENDPOINT)
        protected Notifier endpointNotifier;

        @Autowired
        @Qualifier(Notifier.SOURCE)
        protected Notifier sourceNotifier;

        @Bean(Notifier.FEEDBACK)
        @ConditionalOnMissingBean(name = Notifier.FEEDBACK)
        public FeedbackNotifier feedbackNotifier() throws Exception {
            FeedbackNotifier feedbackNotifier = new FeedbackNotifier();
            BeanUtils.copyProperties(this, feedbackNotifier);
            log.info("FeedbackNotifier inited");
            return feedbackNotifier;
        }
    }
}
