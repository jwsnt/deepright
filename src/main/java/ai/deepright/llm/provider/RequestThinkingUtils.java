package ai.deepright.llm.provider;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.commons.lang3.StringUtils;

public class RequestThinkingUtils {

    public static void notifyMessage(NotifierService notifierService, WorkflowTask workTask, String thinking) throws Exception {
        if (!StringUtils.isEmpty(thinking)) {
            notifierService.notify(Segment.build(workTask, Segment.SegmentConfig.builder()
                    .workflow(ProviderRequestService.KEY_THINKING)
                    .content(new StringBuffer(thinking))
                    .notifier(Notifier.SOURCE)
                    .build()), workTask, workTask);
        }
    }
}
