package ai.open.right.workflow.flow.llm.store.history;

import ai.open.right.workflow.flow.summary.SummaryConfig;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;

public class HistoryConfigTest {

    @Test
    public void testHasClear() {
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        historyClearConfig.setRepositories(Arrays.asList("HELLO WORLD"));
        HistoryConfig historyConfig = new HistoryConfig();
        historyConfig.setClearConfig(historyClearConfig);
        Assert.assertEquals("[HELLO WORLD]", historyConfig.getClearConfig().getRepositories().toString());
        Assert.assertTrue(historyConfig.hasClear());
    }

    @Test
    public void testMergeBothNull() throws Exception {
        HistoryConfig config1 = new HistoryConfig();
        HistoryConfig config2 = new HistoryConfig();
        HistoryConfig merged = config1.merge(config2);
        Assert.assertNull(merged.getClearConfig());
        Assert.assertNull(merged.getSummaryConfig());
    }

    @Test
    public void testMergeConfig1Null() throws Exception {
        HistoryConfig config1 = new HistoryConfig();
        HistoryConfig config2 = new HistoryConfig();
        HistoryClearConfig clearConfig = new HistoryClearConfig();
        clearConfig.setRepositories(Arrays.asList("repo1"));
        config2.setClearConfig(clearConfig);
        SummaryConfig summaryConfig = new SummaryConfig();
        config2.setSummaryConfig(summaryConfig);
        HistoryConfig merged = config1.merge(config2);
        Assert.assertEquals("[repo1]", merged.getClearConfig().getRepositories().toString());
        Assert.assertNotNull(merged.getSummaryConfig());
    }

    @Test
    public void testMergeConfig2Null() throws Exception {
        HistoryConfig config1 = new HistoryConfig();
        HistoryConfig config2 = new HistoryConfig();
        HistoryClearConfig clearConfig = new HistoryClearConfig();
        clearConfig.setRepositories(Arrays.asList("repo2"));
        config1.setClearConfig(clearConfig);
        SummaryConfig summaryConfig = new SummaryConfig();
        config1.setSummaryConfig(summaryConfig);
        HistoryConfig merged = config1.merge(config2);
        Assert.assertEquals("[repo2]", merged.getClearConfig().getRepositories().toString());
        Assert.assertNotNull(merged.getSummaryConfig());
    }

    @Test
    public void testMergeBothHaveValues() throws Exception {
        HistoryConfig config1 = new HistoryConfig();
        HistoryConfig config2 = new HistoryConfig();
        HistoryClearConfig clear1 = new HistoryClearConfig();
        clear1.setRepositories(Arrays.asList("repo1"));
        config1.setClearConfig(clear1);
        HistoryClearConfig clear2 = new HistoryClearConfig();
        clear2.setRepositories(Arrays.asList("repo2"));
        config2.setClearConfig(clear2);
        SummaryConfig summary1 = new SummaryConfig();
        config1.setSummaryConfig(summary1);
        SummaryConfig summary2 = new SummaryConfig();
        config2.setSummaryConfig(summary2);
        HistoryConfig merged = config1.merge(config2);
        Assert.assertNotNull(merged.getClearConfig());
        Assert.assertNotNull(merged.getSummaryConfig());
    }

    @Test
    public void testMergePartialNulls() throws Exception {
        HistoryConfig config1 = new HistoryConfig();
        HistoryConfig config2 = new HistoryConfig();
        HistoryClearConfig clear1 = new HistoryClearConfig();
        clear1.setRepositories(Arrays.asList("repo1"));
        config1.setClearConfig(clear1);
        SummaryConfig summary2 = new SummaryConfig();
        config2.setSummaryConfig(summary2);
        HistoryConfig merged = config1.merge(config2);
        Assert.assertEquals("[repo1]", merged.getClearConfig().getRepositories().toString());
        Assert.assertNotNull(merged.getSummaryConfig());
    }
}
