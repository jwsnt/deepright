package ai.open.right.workflow.flow.tools.remote;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.tools.ToolsConfig;
import ai.open.right.workflow.flow.tools.ToolsHeader;
import org.apache.http.client.methods.HttpGet;
import org.junit.Assert;
import org.junit.Test;
import java.util.Arrays;

public class ToolsServiceImplTest {
    @Test
    public void testBuildHeaders() throws Exception {
        ToolsServiceImpl service = new ToolsServiceImpl();
        ToolsConfig config = new ToolsConfig();
        ToolsHeader header = new ToolsHeader();
        header.setKey("K");
        header.setVal("V");
        config.setHeaders(Arrays.asList(header));
        HttpGet get = new HttpGet("http://x");
        service.buildHeaders(get, config, ObjectBuilder.buildWorkflowTask());
        Assert.assertEquals("V", get.getFirstHeader("K").getValue());
    }
}
