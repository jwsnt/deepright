package ai.open.right;

import ai.open.right.protocol.ProtocolCode;

public class TakeoverException extends WorkflowException {

    public final static TakeoverException SIGNAL = new TakeoverException();

    private TakeoverException() {
        super("", ProtocolCode.I001);
    }

    @Override
    public TakeoverException needSilent() {
        return this;
    }
}
