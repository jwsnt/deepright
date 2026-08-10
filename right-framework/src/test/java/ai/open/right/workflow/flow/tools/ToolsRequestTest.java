package ai.open.right.workflow.flow.tools;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class ToolsRequestTest {

    @Test
    public void testInit() {
        ToolsRequest toolsRequest = new ToolsRequest();
        toolsRequest.setBiz("BIZ");
        toolsRequest.setChat("CHAT");
        toolsRequest.setConversation("CONV");
        toolsRequest.setTrace("TRACE");
        toolsRequest.setTimestamp(10086L);
        toolsRequest.setWorkflow("WORKFLOW");
        toolsRequest.setQuery("QUERY");
        UserContext userContext = ObjectBuilder.buildLLMQuery().getUserContext();
        toolsRequest.setUserContext(userContext);
        Map<String, Object> metadata = new HashMap<>();
        toolsRequest.setMetadata(metadata);
        Assert.assertEquals("BIZ", toolsRequest.getBiz());
        Assert.assertEquals("CHAT", toolsRequest.getChat());
        Assert.assertEquals("CONV", toolsRequest.getConversation());
        Assert.assertEquals("TRACE", toolsRequest.getTrace());
        Assert.assertEquals(Long.valueOf(10086L), toolsRequest.getTimestamp());
        Assert.assertEquals("WORKFLOW", toolsRequest.getWorkflow());
        Assert.assertEquals("QUERY", toolsRequest.getQuery());
        Assert.assertEquals(userContext, toolsRequest.getUserContext());
        Assert.assertEquals(metadata, toolsRequest.getMetadata());
    }
}
