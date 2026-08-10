package ai.open.right.workflow.notify.impl;

import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.workflow.notify.Notifier;

public class FeedbackNotifierInitConfigTest {

    @Test
    public void shouldCreateFeedbackNotifierWithProvidedProperties() throws Exception {
        FeedbackNotifier.InitConfig init = new FeedbackNotifier.InitConfig();

        Notifier localhostNotifier = EasyMock.createMock(Notifier.class);
        Notifier sourcentNotifier = EasyMock.createMock(Notifier.class);

        EasyMock.replay(localhostNotifier, sourcentNotifier);

        // 设置属性
        init.setLocalhostNotifier(localhostNotifier);
        init.setSourceNotifier(sourcentNotifier);

        FeedbackNotifier bean = init.feedbackNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof FeedbackNotifier);

        EasyMock.verify(localhostNotifier, sourcentNotifier);
    }

    @Test
    public void shouldCreateFeedbackNotifierWithDefaults() throws Exception {
        FeedbackNotifier.InitConfig init = new FeedbackNotifier.InitConfig();

        FeedbackNotifier bean = init.feedbackNotifier();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof FeedbackNotifier);
    }
}
