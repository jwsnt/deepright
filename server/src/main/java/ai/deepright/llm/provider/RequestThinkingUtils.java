package ai.deepright.llm.provider;

import ai.deepright.feature.FeatureFlag;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierService;
import org.apache.commons.lang3.StringUtils;

public class RequestThinkingUtils {

    public static void notifyMessage(NotifierService notifierService, WorkflowTask workTask, StringBuffer thinking) throws Exception {
        if (!FeatureFlag.isSilent(workTask) && !StringUtils.isEmpty(thinking)) {
            notifierService.notify(Segment.build(workTask, Segment.SegmentConfig.builder()
                    .workflow(RequestThinkingUtils.buildWorkflow(workTask, thinking.toString()))
                    .content(new StringBuffer(thinking))
                    .notifier(Notifier.SOURCE)
                    .build()), workTask, workTask);
        }
    }

    // 不同Thinking不合并
    public static String buildWorkflow(WorkflowTask workTask, String thinking) throws Exception {
        return ProviderRequestService.KEY_THINKING + "_" + workTask.getDeepness();
    }
}
