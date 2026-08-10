package ai.open.right.netty.chat.server.http;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class NettyHttpMessageTest {

    @Test
    public void testSettersAndGetters() {
        NettyHttpMessage message = new NettyHttpMessage();
        message.setContent("hello");
        message.setRole("user");
        Assertions.assertEquals("hello", message.getContent());
        Assertions.assertEquals("user", message.getRole());
    }

    @Test
    public void testDefaultRole() {
        NettyHttpMessage message = new NettyHttpMessage();
        Assertions.assertEquals("assistant", message.getRole());
    }
}
