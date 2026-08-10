package ai.open.right.workflow.notify.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
public class NotifierServiceImpl implements NotifierService {

    protected Map<String, Notifier> notifier;

    @Override
    public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        Notifier actual = this.notifier.get(notifier);
        Assert.notNull(actual, "The notifier can not be empty: " + notifier);
        actual.notify(segment, redirectContext, notifierWriteBack, mediaContext);
    }

    @Override
    public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
        Notifier actual = this.notifier.get(notifier);
        Assert.notNull(actual, "The notifier can not be empty: " + notifier);
        actual.notify(segment, redirectContext, notifierWriteBack);
    }

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        Segment.SegmentChecker.check(segment);
        Notifier notifier = this.notifier.get(segment.getNotifier());
        Assert.notNull(notifier, "The notifier can not be empty: " + segment.getNotifier());
        notifier.notify(segment, redirectContext, notifierWriteBack, mediaContext);
    }

    @Override
    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
        Segment.SegmentChecker.check(segment);
        Notifier notifier = this.notifier.get(segment.getNotifier());
        Assert.notNull(notifier, "The notifier can not be empty: " + segment.getNotifier());
        notifier.notify(segment, redirectContext, notifierWriteBack);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack, mediaContext);
    }

    @Override
    // 特殊用途，不传递Deepness
    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
        this.notify(segment, RedirectContext.EMPTY, notifierWriteBack);
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected Map<String, Notifier> notifier;

        @Bean
        @ConditionalOnMissingBean(value = NotifierService.class)
        public NotifierService notifierService() throws Exception {
            NotifierServiceImpl notifierService = new NotifierServiceImpl();
            BeanUtils.copyProperties(this, notifierService);
            log.info("NotifierServiceImpl inited: notifier={}", notifierService.getNotifier());
            return notifierService;
        }
    }
}
