package ai.open.right.workflow.notify;

import ai.open.right.context.RedirectContext;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.llm.Segment;

import java.util.List;

public interface Notifier {

    public static final String LOCALHOST = "localhost";

    public static final String ENDPOINT = "endpoint";

    public static final String FEEDBACK = "feedback";

    public static final String SOURCE = "source";

    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception;

    public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception;

    public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception;

    public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception;
}
