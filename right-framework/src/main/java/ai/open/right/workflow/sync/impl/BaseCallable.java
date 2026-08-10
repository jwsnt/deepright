package ai.open.right.workflow.sync.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncCallable;

public class BaseCallable implements SyncCallable {

    @Override
    public BaseCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
        return this;
    }

    @Override
    public BaseCallable setRedirectContext(RedirectContext redirectContext) {
        return this;
    }

    @Override
    public BaseCallable setNotifierService(NotifierService notifierService) {
        return this;
    }

    @Override
    public void call(Segment segment) throws Exception {

    }
}
