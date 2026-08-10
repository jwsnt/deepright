package ai.open.right.workflow.sync.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import ai.open.right.workflow.notify.NotifierWriteBack;
import org.junit.Assert;
import org.junit.Test;

public class NotifierCallableTest {

    @Test
    public void test() throws Exception {
        NotifierServiceImpl _notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        NotifierWriteBack _notifyWriteBack = ObjectBuilder.buildNotifyWriteBack();
        RedirectContext _redirectContext = RedirectContext.EMPTY;
        Segment _segment = new SegmentDelegate(ObjectBuilder.buildLLMQuery());
        BaseCallable baseCallable = new BaseCallable() {
            @Override
            public BaseCallable setRedirectContext(RedirectContext redirectContext) {
                Assert.assertEquals(_redirectContext, redirectContext);
                return this;
            }

            @Override
            public BaseCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
                Assert.assertEquals(_notifyWriteBack, notifierWriteBack);
                return this;
            }

            @Override
            public BaseCallable setNotifierService(NotifierService notifierService) {
                Assert.assertEquals(_notifierManager, notifierService);
                return this;
            }

            @Override
            public void call(Segment segment) {
                Assert.assertEquals(_segment, segment);
            }
        };
        NotifierCallable notifierCallable = new NotifierCallable(new BaseCallable(), "HELLO");
        notifierCallable.setNotifierService(_notifierManager);
        notifierCallable.setRedirectContext(_redirectContext);
        notifierCallable.setNotifierWriteBack(_notifyWriteBack);
        notifierCallable.call(_segment);
    }
}
