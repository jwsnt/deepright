package ai.open.right.workflow.notify;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.media.MediaContext;

import java.util.List;

public interface NotifierService {

    public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception;

    public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception;

    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception;

    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception;

    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception;

    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception;
}
