package ai.open.right.workflow.flow.llm.provider.google;

import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class VertexTokenExchangeInitConfigTest {

    @Test
    public void testInit() throws Exception {
        CloseableHttpAsyncClient other = EasyMock.createMock(CloseableHttpAsyncClient.class);
        EasyMock.replay(other);
        VertexTokenExchange.InitConfig initConfig = new VertexTokenExchange.InitConfig();
        initConfig.setOther(other);
        initConfig.setProject("PROJECT");
        initConfig.setRemote("REMOTE");
        initConfig.setAppId("APPID");
        VertexTokenExchange empty = initConfig.vertexTokenExchange();
        Assert.assertEquals(other, empty.getOther());
        Assert.assertEquals("PROJECT", empty.getProject());
        Assert.assertEquals("REMOTE", empty.getRemote());
        Assert.assertEquals("APPID", empty.getAppId());
    }
}
