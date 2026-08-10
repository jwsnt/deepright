package ai.open.right.workflow.flow.llm.store.history;

import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.List;

public class HistoryClearConfigTest {

    @Test
    public void test() {
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        historyClearConfig.setOffset(1000);
        Assert.assertEquals(Integer.valueOf(1000), historyClearConfig.getOffset());
        Assert.assertEquals(Long.valueOf(20086L - 1000), historyClearConfig.getOffset(20086L));
    }

    @Test
    public void testWithNotOffset() {
        HistoryClearConfig historyClearConfig = new HistoryClearConfig();
        Assert.assertNull(historyClearConfig.getOffset());
        Assert.assertEquals(Long.valueOf(20086L), historyClearConfig.getOffset(20086L));
    }

    @Test
    public void testMergeBothNull() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        HistoryClearConfig other = null;
        config.merge(other);
        Assert.assertNull(config.getRepositories());
        Assert.assertNull(config.getOffset());
    }

    @Test
    public void testMergeThisHasValues() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        config.setRepositories(List.of("repo1"));
        config.setOffset(100);
        HistoryClearConfig other = new HistoryClearConfig();
        other.setRepositories(Arrays.asList("repo2"));
        other.setOffset(200);
        config.merge(other);
        Assert.assertEquals("repo1", config.getRepositories().getFirst());
        Assert.assertEquals("repo2", config.getRepositories().getLast());
        Assert.assertEquals(Integer.valueOf(100), config.getOffset());
    }

    @Test
    public void testMergeOtherHasValues() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        HistoryClearConfig other = new HistoryClearConfig();
        other.setRepositories(Arrays.asList("repo2"));
        other.setOffset(200);
        config.merge(other);
        Assert.assertEquals(Arrays.asList("repo2"), config.getRepositories());
        Assert.assertEquals(Integer.valueOf(200), config.getOffset());
    }

    @Test
    public void testMergePartialValues() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        config.setRepositories(Arrays.asList("repo1"));
        HistoryClearConfig other = new HistoryClearConfig();
        other.setOffset(200);
        config.merge(other);
        Assert.assertEquals(Arrays.asList("repo1"), config.getRepositories());
        Assert.assertEquals(Integer.valueOf(200), config.getOffset());
    }

    @Test
    public void testMergeOtherRepositoriesNull() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        HistoryClearConfig other = new HistoryClearConfig();
        other.setOffset(200);
        config.merge(other);
        Assert.assertNull(config.getRepositories());
        Assert.assertEquals(Integer.valueOf(200), config.getOffset());
    }

    @Test
    public void testMergeOtherOffsetNull() throws Exception {
        HistoryClearConfig config = new HistoryClearConfig();
        HistoryClearConfig other = new HistoryClearConfig();
        other.setRepositories(Arrays.asList("repo2"));
        config.merge(other);
        Assert.assertEquals(Arrays.asList("repo2"), config.getRepositories());
        Assert.assertNull(config.getOffset());
    }
}
