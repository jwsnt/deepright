package ai.open.right.workflow.flow.llm.store.history;

import org.junit.Assert;
import org.junit.Test;

public class HistoryFetchConfigTest {

    @Test
    public void test() {
        HistoryFetchConfig historyFetchConfig = new HistoryFetchConfig();
        Assert.assertEquals(HistoryFetchConfig.NUMS, historyFetchConfig.getNums());
        historyFetchConfig.setNums(1000);
        Assert.assertEquals(Integer.valueOf(1000), historyFetchConfig.getNums());
        historyFetchConfig.setScene("A");
        Assert.assertEquals("A", historyFetchConfig.getScene());
    }

    @Test
    public void testMerge() throws Exception {
        HistoryFetchConfig historyFetchConfig1 = new HistoryFetchConfig();
        historyFetchConfig1.setNums(1000);
        historyFetchConfig1.setScene("X");
        HistoryFetchConfig historyFetchConfig2 = new HistoryFetchConfig();
        historyFetchConfig2.setScene("Y");
        historyFetchConfig2.merge(historyFetchConfig1);
        Assert.assertEquals("Y", historyFetchConfig2.getScene());
        Assert.assertEquals(Integer.valueOf(1000), historyFetchConfig2.getNums());
    }
}
