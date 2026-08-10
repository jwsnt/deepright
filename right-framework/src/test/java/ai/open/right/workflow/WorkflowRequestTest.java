package ai.open.right.workflow;

import ai.open.right.context.UserContext;
import ai.open.right.netty.chat.distribute.NettyRequest;
import org.junit.Test;

import java.util.HashMap;

public class WorkflowRequestTest {

    @Test
    public void testCheck() {
        NettyRequest workflowRequest = new NettyRequest();
        workflowRequest.setProtocol("full");
        workflowRequest.setWorkflow("workflow");
        workflowRequest.setChat("chat");
        workflowRequest.setBiz("biz");
        workflowRequest.setTrace("trace");
        UserContext userContext = UserContext.builder().build();
        userContext.setSystem("X");
        userContext.setRegion("D");
        userContext.setModel("FG");
        userContext.setLanguage("4");
        userContext.setBrand("5");
        userContext.setDevice("7");
        workflowRequest.setUserContext(userContext);
        workflowRequest.setConversation("conversation");
        workflowRequest.setMetadata(new HashMap<>());
        workflowRequest.setCreated(System.currentTimeMillis());
        NettyRequest.NettyRequestChecker.check(workflowRequest);
    }
}
