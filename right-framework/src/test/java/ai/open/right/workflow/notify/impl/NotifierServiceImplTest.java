package ai.open.right.workflow.notify.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

public class NotifierServiceImplTest {

    @Test
    public void testNotify() throws Exception {
        Segment segment = ObjectBuilder.buildSegment();
        segment.setNotifier("test-notifier");
        Notifier notifier = EasyMock.createMock(Notifier.class);
        RedirectContext redirectContext = RedirectContext.EMPTY;
        NotifierWriteBack notifierWriteBack = EasyMock.createMock(NotifierWriteBack.class);
        notifier.notify(segment, redirectContext, notifierWriteBack);
        EasyMock.expectLastCall().once();
        EasyMock.replay(notifier, notifierWriteBack);
        NotifierServiceImpl service = new NotifierServiceImpl();
        Map<String, Notifier> notifiers = new HashMap<>();
        notifiers.put("test-notifier", notifier);
        service.setNotifier(notifiers);
        service.notify(segment, redirectContext, notifierWriteBack);
        EasyMock.verify(notifier, notifierWriteBack);
    }

    @Test
    public void testInitConfig() throws Exception {
        NotifierServiceImpl.InitConfig config = new NotifierServiceImpl.InitConfig();
        Map<String, Notifier> notifiers = new HashMap<>();
        config.setNotifier(notifiers);
        Assertions.assertNotNull(config.notifierService());
    }
}
