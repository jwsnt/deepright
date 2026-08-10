package ai.open.right.workflow.sync.impl;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.sync.SyncCallable;

public class NotifierCallable implements SyncCallable {

    protected NotifierWriteBack notifierWriteBack;

    protected NotifierService notifierService;

    protected RedirectContext redirectContext;

    protected final SyncCallable syncCallable;

    protected final String notifier;

    public NotifierCallable(SyncCallable syncCallable, String notifier) {
        this.syncCallable = syncCallable;
        this.notifier = notifier;
    }

    public NotifierCallable(String notifier) {
        this(null, notifier);
    }

    @Override
    public SyncCallable setRedirectContext(RedirectContext redirectContext) {
        this.redirectContext = redirectContext;
        if (this.syncCallable != null) {
            return this.syncCallable.setRedirectContext(this.redirectContext);
        }
        return this;
    }

    @Override
    public SyncCallable setNotifierWriteBack(NotifierWriteBack notifierWriteBack) {
        this.notifierWriteBack = notifierWriteBack;
        if (this.syncCallable != null) {
            return this.syncCallable.setNotifierWriteBack(this.notifierWriteBack);
        }
        return this;
    }

    @Override
    public SyncCallable setNotifierService(NotifierService notifierService) {
        this.notifierService = notifierService;
        if (this.syncCallable != null) {
            return this.syncCallable.setNotifierService(this.notifierService);
        }
        return this;
    }

    @Override
    public void call(Segment segment) throws Exception {
        this.notifierService.notify(segment.copyWithNotifier(this.notifier), this.redirectContext, this.notifierWriteBack);
        if (this.syncCallable != null) {
            this.syncCallable.call(segment);
        }
    }
}
