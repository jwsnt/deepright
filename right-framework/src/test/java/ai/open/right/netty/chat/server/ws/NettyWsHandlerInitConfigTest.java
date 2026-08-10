package ai.open.right.netty.chat.server.ws;

import ai.open.right.netty.chat.NettyChatHandler;
import ai.open.right.netty.chat.distribute.NettyDistributor;
import org.junit.Assert;
import org.junit.Test;

public class NettyWsHandlerInitConfigTest {

    @Test
    public void initConfig_autodumpGetterSetter() {
        NettyWsHandler.InitConfig init = new NettyWsHandler.InitConfig();
        Assert.assertNull(init.getAutoDump());

        init.setAutoDump("/tmp/netty-ws-harness-test");
        Assert.assertEquals("/tmp/netty-ws-harness-test", init.getAutoDump());

        init.setAutoDump(null);
        Assert.assertNull(init.getAutoDump());
    }

    @Test
    public void shouldCreateNettyWsHandlerWithProvidedProperties() throws Exception {
        NettyWsHandler.InitConfig init = new NettyWsHandler.InitConfig();

        NettyDistributor distributor = new NettyDistributor();

        init.setDistributor(distributor);
        init.setAutoDump("/tmp/netty-ws-harness-test");

        NettyWsHandler bean = init.nettyWsHandler();

        Assert.assertSame(distributor, bean.getDistributor());
        Assert.assertEquals("/tmp/netty-ws-harness-test", bean.getAutoDump());

        try {
            java.lang.reflect.Field field = NettyChatHandler.class.getDeclaredField("distributor");
            field.setAccessible(true);
            Object actual = field.get(bean);
            Assert.assertSame(distributor, actual);
        } catch (Exception e) {
            throw new AssertionError(e);
        }
    }

    @Test
    public void shouldCreateNettyWsHandlerWithDefaults() throws Exception {
        NettyWsHandler.InitConfig init = new NettyWsHandler.InitConfig();

        NettyWsHandler bean = init.nettyWsHandler();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof NettyWsHandler);
        Assert.assertNull(bean.getAutoDump());

        try {
            java.lang.reflect.Field field = NettyChatHandler.class.getDeclaredField("distributor");
            field.setAccessible(true);
            Object actual = field.get(bean);
            Assert.assertNull(actual);
        } catch (Exception e) {
            throw new AssertionError(e);
        }
    }
}
