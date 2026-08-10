package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.llm.signal.SignalRemoval;
import org.junit.Assert;
import org.junit.Test;

public class SignalRemovalTest {

    @Test
    public void test() {
        SignalRemoval removal = new SignalStreamImpl();
        String target = removal.remove("${I_01;S_00;S_02;SUG_ITEM_CARD=100020} 好的，您想购买4GB的流量包，我们有4GB 30天，4GB 7天，4GB 1天，您想要哪一款呢？");
        Assert.assertEquals(" 好的，您想购买4GB的流量包，我们有4GB 30天，4GB 7天，4GB 1天，您想要哪一款呢？", target);
    }
}
