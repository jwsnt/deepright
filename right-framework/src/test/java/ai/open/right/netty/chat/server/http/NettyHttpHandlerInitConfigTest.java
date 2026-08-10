package ai.open.right.netty.chat.server.http;

import ai.open.right.netty.chat.NettyChatHandler;
import org.junit.Assert;
import org.junit.Test;

import ai.open.right.netty.chat.distribute.NettyDistributor;

public class NettyHttpHandlerInitConfigTest {

    @Test
    public void shouldCreateNettyHttpHandlerWithProvidedProperties() throws Exception {
        NettyHttpHandler.InitConfig init = new NettyHttpHandler.InitConfig();

        NettyDistributor distributor = new NettyDistributor();

        // setter 注入
        init.setDistributor(distributor);
        init.setAutoDump("/tmp/netty-http-harness-test");

        NettyHttpHandler bean = init.nettyHttpHandler();

        Assert.assertSame(distributor, bean.getDistributor());
        Assert.assertEquals("/tmp/netty-http-harness-test", bean.getAutoDump());

        // 通过反射验证属性复制
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
    public void shouldCreateNettyHttpHandlerWithDefaultsWhenNoPropertiesProvided() throws Exception {
        NettyHttpHandler.InitConfig init = new NettyHttpHandler.InitConfig();

        NettyHttpHandler bean = init.nettyHttpHandler();

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

 