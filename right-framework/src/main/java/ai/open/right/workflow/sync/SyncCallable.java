package ai.open.right.workflow.sync;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;

public interface SyncCallable {

    public SyncCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack);

    public SyncCallable setNotifierService(NotifierService notifierService);

    public SyncCallable setRedirectContext(RedirectContext redirectContext);

    public void call(Segment segment) throws Exception;
}
