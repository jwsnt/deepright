package ai.open.right.workflow.flow.llm;

import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.notify.Notifier;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

public interface LLMQuery extends WorkflowTask {

    public void callToLocalHost();

    public void callToEndpoint();

    public static class LLMQueryChecker {

        public static void check(LLMQuery query) {
            Assert.hasText(query.getConversation(), "Conversation can not be empty");
            Assert.notNull(query.getUserContext(), "User Context can not be empty");
            Assert.notNull(query.getCreated(), "Timestamp can not be empty");
            Assert.hasText(query.getWorkflow(), "Workflow can not be empty");
            Assert.hasText(query.getNotifier(), "Notifier can not be empty");
            Assert.notNull(query.getQuery(), "Query can not be empty");
            Assert.notNull(query.getChat(), "Chat can not be empty");
            Assert.hasText(query.getBiz(), "Biz can not be empty");
            UserContext.UserContextChecker.check(query.getUserContext());
        }
    }


    public static LLMQuery build(WorkflowTask task, String workflow, String notifier) {
        return new LLMQueryDelegate(task, StringUtils.hasText(workflow) ? workflow : DefaultAssistant.WORKFLOW_NAME, notifier);
    }

    public static LLMQuery build(WorkflowTask task, String workflow) {
        return LLMQuery.build(task, workflow, Notifier.ENDPOINT);
    }

    public static LLMQuery build(WorkflowTask task) {
        return LLMQuery.build(task, task.getWorkflow(), Notifier.ENDPOINT);
    }
}

