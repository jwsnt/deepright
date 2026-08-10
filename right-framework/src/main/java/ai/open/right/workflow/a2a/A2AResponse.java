package ai.open.right.workflow.a2a;

import ai.open.right.netty.NettyStream;

public interface A2AResponse extends NettyStream {

    public String getJsonrpc();

    // Message | Task
    public Object getResult();

    public Object getId();
}
