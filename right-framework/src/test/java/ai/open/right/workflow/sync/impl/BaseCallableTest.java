package ai.open.right.workflow.sync.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.SegmentDelegate;
import org.junit.Test;

public class BaseCallableTest {
    @Test
    public void test() throws Exception {
        BaseCallable baseCallable = new BaseCallable();
        baseCallable.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithNothing());
        baseCallable.setRedirectContext(RedirectContext.EMPTY);
        baseCallable.setNotifierWriteBack(ObjectBuilder.buildNotifyWriteBack());
        baseCallable.call(new SegmentDelegate(ObjectBuilder.buildWorkflowTask()));
    }
}
