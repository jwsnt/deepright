package ai.open.right.workflow.config;

import org.junit.Assert;
import org.junit.Test;

public class ConfigSearchTest {

    @Test
    public void testSetGet() {
        ConfigSearch configSearch = new ConfigSearch();
        configSearch.setBiz("BIZ");
        configSearch.setLanguage("CH");
        configSearch.setDevice("DE");
        Assert.assertEquals("BIZ", configSearch.getBiz());
        Assert.assertEquals("CH", configSearch.getLanguage());
        Assert.assertEquals("DE", configSearch.getDevice());
        Assert.assertNotNull(configSearch.toString());
    }

    @Test
    public void testBuild() {
        ConfigSearch configSearch = ConfigSearch.builder()
                .biz("BIZ").language("CH").device("DE")
                .build();
        Assert.assertEquals("BIZ", configSearch.getBiz());
        Assert.assertEquals("CH", configSearch.getLanguage());
        Assert.assertEquals("DE", configSearch.getDevice());
    }

    @Test
    public void testInit() {
        ConfigSearch configSearch = new ConfigSearch("BIZ", "CH", "DE");
        Assert.assertEquals("BIZ", configSearch.getBiz());
        Assert.assertEquals("DE", configSearch.getLanguage());
        Assert.assertEquals("CH", configSearch.getDevice());
    }
}
