package ai.open.right.netty.chat.server.http;

import lombok.Getter;
import lombok.Setter;

@Setter
@Getter
// Netty Open AI 请求
public class NettyHttpMessage {

    protected String content;

    protected String role = "assistant";
}