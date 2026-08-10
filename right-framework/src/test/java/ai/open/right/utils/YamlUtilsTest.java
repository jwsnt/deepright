package ai.open.right.utils;

import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;

public class YamlUtilsTest {

    @Test
    public void test() throws Exception {
        Map<String, Object> yaml = YamlUtils.read(IOUtils.toString(ResourceUtils.getURL("classpath:skills/fullskill/SKILL.md").openStream(), StandardCharsets.UTF_8));
        Assert.assertEquals("pdf-processing", yaml.get("name"));
        Assert.assertEquals("Extract text and tables from PDF files, fill forms, merge documents.", yaml.get("description"));
        Assert.assertEquals("Bash(git:*) Bash(jq:*) Read", yaml.get("allowed-tools"));
        Assert.assertEquals("1.0", Map.class.cast(yaml.get("metadata")).get("version"));
        Assert.assertEquals("example-org", Map.class.cast(yaml.get("metadata")).get("xd"));
        Assert.assertEquals("Designed for Claude Code (or similar products)", yaml.get("compatibility"));
        String string = YamlUtils.write(yaml);
        Assert.assertEquals("allowed-tools: \"Bash(git:*) Bash(jq:*) Read\"\n" +
                "compatibility: \"Designed for Claude Code (or similar products)\"\n" +
                "description: \"Extract text and tables from PDF files, fill forms, merge documents.\"\n" +
                "license: \"Apache-2.0\"\n" +
                "metadata:\n" +
                "  version: \"1.0\"\n" +
                "  xd: \"example-org\"\n" +
                "name: \"pdf-processing\"\n", string);
    }

    @Test
    public void test2() throws Exception {
        Map<String, Object> yaml = YamlUtils.read(IOUtils.toString(ResourceUtils.getURL("classpath:skills/fullskill/SKILL.md").openStream(), StandardCharsets.UTF_8));
        Assert.assertEquals("pdf-processing", yaml.get("name"));
        Assert.assertEquals("Extract text and tables from PDF files, fill forms, merge documents.", yaml.get("description"));
        Assert.assertEquals("Bash(git:*) Bash(jq:*) Read", yaml.get("allowed-tools"));
        Assert.assertEquals("1.0", Map.class.cast(yaml.get("metadata")).get("version"));
        Assert.assertEquals("example-org", Map.class.cast(yaml.get("metadata")).get("xd"));
        Assert.assertEquals("Designed for Claude Code (or similar products)", yaml.get("compatibility"));
        String string = YamlUtils.instance().writeValueAsString(yaml);
        Assert.assertEquals("allowed-tools: \"Bash(git:*) Bash(jq:*) Read\"\n" +
                "compatibility: \"Designed for Claude Code (or similar products)\"\n" +
                "description: \"Extract text and tables from PDF files, fill forms, merge documents.\"\n" +
                "license: \"Apache-2.0\"\n" +
                "metadata:\n" +
                "  version: \"1.0\"\n" +
                "  xd: \"example-org\"\n" +
                "name: \"pdf-processing\"\n", string);
    }

    @Test
    public void test3() throws Exception {
        Map<String, Object> yaml = YamlUtils.read(ResourceUtils.getURL("classpath:skills/fullskill/SKILL.md").openStream());
        Assert.assertEquals("pdf-processing", yaml.get("name"));
        Assert.assertEquals("Extract text and tables from PDF files, fill forms, merge documents.", yaml.get("description"));
        Assert.assertEquals("Bash(git:*) Bash(jq:*) Read", yaml.get("allowed-tools"));
        Assert.assertEquals("1.0", Map.class.cast(yaml.get("metadata")).get("version"));
        Assert.assertEquals("example-org", Map.class.cast(yaml.get("metadata")).get("xd"));
        Assert.assertEquals("Designed for Claude Code (or similar products)", yaml.get("compatibility"));
        String string = YamlUtils.instance().writeValueAsString(yaml);
        Assert.assertEquals("allowed-tools: \"Bash(git:*) Bash(jq:*) Read\"\n" +
                "compatibility: \"Designed for Claude Code (or similar products)\"\n" +
                "description: \"Extract text and tables from PDF files, fill forms, merge documents.\"\n" +
                "license: \"Apache-2.0\"\n" +
                "metadata:\n" +
                "  version: \"1.0\"\n" +
                "  xd: \"example-org\"\n" +
                "name: \"pdf-processing\"\n", string);
    }

    /**
     * 覆盖 read(InputStream) 的 return null 分支：入参为 null 时返回 null
     */
    @Test
    public void testReadInputStreamNull() throws Exception {
        Map<String, Object> result = YamlUtils.read((InputStream) null);
        Assert.assertNull(result);
    }

    /**
     * 覆盖 read(String) 的 return null 分支：入参为 null 时返回 null
     */
    @Test
    public void testReadStringNull() throws Exception {
        Map<String, Object> result = YamlUtils.read((String) null);
        Assert.assertNull(result);
    }

    /**
     * 覆盖 write(Object) 的 return null 分支：入参为 null 时返回 null
     */
    @Test
    public void testWriteNull() throws Exception {
        String result = YamlUtils.write(null);
        Assert.assertNull(result);
    }
}
