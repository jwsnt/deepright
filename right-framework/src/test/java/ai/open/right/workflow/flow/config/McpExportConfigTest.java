package ai.open.right.workflow.flow.config;

import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class McpExportConfigTest {

    @Test
    public void testSetGet() {
        Map<String, Object> properties = new HashMap<>();
        List<String> methods = Arrays.asList("A", "B");
        List<String> require = Arrays.asList("A", "B");
        McpExportConfig mcpExportConfig = new McpExportConfig();
        Assert.assertFalse(mcpExportConfig.hasQuery());
        mcpExportConfig.setBiz("BIZ");
        mcpExportConfig.setWorkflow("WORKFLOW");
        mcpExportConfig.setName("NAME");
        mcpExportConfig.setDescription("DESCRIPTION");
        mcpExportConfig.setMethods(methods);
        mcpExportConfig.setQuery("QUERY");
        mcpExportConfig.setProperties(properties);
        mcpExportConfig.setMimeType("MIME");
        mcpExportConfig.setUri("URI");
        mcpExportConfig.setRequired(require);
        mcpExportConfig.setUriTemplate("URITemplate");
        Assert.assertEquals(properties, mcpExportConfig.getProperties());
        Assert.assertEquals(methods, mcpExportConfig.getMethods());
        Assert.assertEquals("BIZ", mcpExportConfig.getBiz());
        Assert.assertEquals("WORKFLOW", mcpExportConfig.getWorkflow());
        Assert.assertEquals("NAME", mcpExportConfig.getName());
        Assert.assertEquals("DESCRIPTION", mcpExportConfig.getDescription());
        Assert.assertEquals("QUERY", mcpExportConfig.getQuery());
        Assert.assertEquals("MIME", mcpExportConfig.getMimeType());
        Assert.assertEquals("URI", mcpExportConfig.getUri());
        Assert.assertEquals(require, mcpExportConfig.getRequired());
        Assert.assertEquals("URITemplate", mcpExportConfig.getUriTemplate());
        Assert.assertTrue(mcpExportConfig.hasQuery());
    }

    @Test
    public void testInMethod() {
        List<String> methods = Arrays.asList("a", "B");
        McpExportConfig mcpExportConfig = new McpExportConfig();
        Assert.assertTrue(mcpExportConfig.inMethod("X"));
        mcpExportConfig.setMethods(methods);
        Assert.assertFalse(mcpExportConfig.inMethod("X"));
        Assert.assertTrue(mcpExportConfig.inMethod("a"));
        Assert.assertFalse(mcpExportConfig.inMethod("b"));
    }

    @Test
    public void testMergeWithNullArg() throws Exception {
        McpExportConfig a = new McpExportConfig();
        a.setDescription("DESC");
        a.setUriTemplate("/t/{id}");
        a.setProperties(new HashMap<>());
        a.setMimeType("application/json");
        a.setWorkflow("WF");
        a.setRequired(Arrays.asList("r1"));
        a.setMethods(Arrays.asList("get"));
        a.setQuery("q");
        a.setName("N");
        a.setUri("/u");
        a.setBiz("B");
        McpExportConfig result = a.merge(null);
        Assert.assertSame(a, result);
        Assert.assertEquals("DESC", a.getDescription());
        Assert.assertEquals("/t/{id}", a.getUriTemplate());
        Assert.assertEquals("application/json", a.getMimeType());
        Assert.assertEquals("WF", a.getWorkflow());
        Assert.assertEquals(Arrays.asList("r1"), a.getRequired());
        Assert.assertEquals(Arrays.asList("get"), a.getMethods());
        Assert.assertEquals("q", a.getQuery());
        Assert.assertEquals("N", a.getName());
        Assert.assertEquals("/u", a.getUri());
        Assert.assertEquals("B", a.getBiz());
    }

    @Test
    public void testMergeCopiesFromOtherWhenBlankOrNull() throws Exception {
        McpExportConfig a = new McpExportConfig();
        a.setDescription("");
        a.setUriTemplate(null);
        a.setProperties(null);
        a.setMimeType("");
        a.setWorkflow("");
        a.setRequired(null);
        a.setMethods(null);
        a.setQuery(null);
        a.setName("");
        a.setUri(null);
        a.setBiz("");
        McpExportConfig b = new McpExportConfig();
        Map<String, Object> props = new HashMap<>();
        props.put("k", "v");
        b.setDescription("DESC2");
        b.setUriTemplate("/t2/{x}");
        b.setProperties(props);
        b.setMimeType("application/xml");
        b.setWorkflow("WF2");
        b.setRequired(Arrays.asList("r2a", "r2b"));
        b.setMethods(Arrays.asList("list", "read"));
        b.setQuery("q2");
        b.setName("N2");
        b.setUri("/u2");
        b.setBiz("B2");
        a.merge(b);
        Assert.assertEquals("DESC2", a.getDescription());
        Assert.assertEquals("/t2/{x}", a.getUriTemplate());
        Assert.assertEquals(props, a.getProperties());
        Assert.assertEquals("application/xml", a.getMimeType());
        Assert.assertEquals("WF2", a.getWorkflow());
        Assert.assertEquals(Arrays.asList("r2a", "r2b"), a.getRequired());
        Assert.assertEquals(Arrays.asList("list", "read"), a.getMethods());
        Assert.assertEquals("q2", a.getQuery());
        Assert.assertEquals("N2", a.getName());
        Assert.assertEquals("/u2", a.getUri());
        Assert.assertEquals("B2", a.getBiz());
    }

    @Test
    public void testMergeDoesNotOverrideNonBlankOrNonNull() throws Exception {
        McpExportConfig a = new McpExportConfig();
        a.setDescription("A");
        a.setUriTemplate("/a/{y}");
        a.setProperties(new HashMap<>());
        a.setMimeType("application/json");
        a.setWorkflow("WF_A");
        a.setRequired(Arrays.asList("ra"));
        a.setMethods(Arrays.asList("get"));
        a.setQuery("qa");
        a.setName("NA");
        a.setUri("/ua");
        a.setBiz("BA");
        McpExportConfig b = new McpExportConfig();
        Map<String, Object> propsB = new HashMap<>();
        propsB.put("k2", "v2");
        b.setDescription("B");
        b.setUriTemplate("/b/{z}");
        b.setProperties(propsB);
        b.setMimeType("text/plain");
        b.setWorkflow("WF_B");
        b.setRequired(Arrays.asList("rb"));
        b.setMethods(Arrays.asList("list"));
        b.setQuery("qb");
        b.setName("NB");
        b.setUri("/ub");
        b.setBiz("BB");
        a.merge(b);
        Assert.assertEquals("A", a.getDescription());
        Assert.assertEquals("/a/{y}", a.getUriTemplate());
        Assert.assertEquals("v2", a.getProperties().get("k2"));
        Assert.assertEquals("application/json", a.getMimeType());
        Assert.assertEquals("WF_A", a.getWorkflow());
        Assert.assertEquals("ra", a.getRequired().getFirst());
        Assert.assertEquals("rb", a.getRequired().getLast());
        Assert.assertEquals("get", a.getMethods().getFirst());
        Assert.assertEquals("list", a.getMethods().getLast());
        Assert.assertEquals("qa", a.getQuery());
        Assert.assertEquals("NA", a.getName());
        Assert.assertEquals("/ua", a.getUri());
        Assert.assertEquals("BA", a.getBiz());
    }

    @Test
    public void testMergeKeepsEmptyCollectionsWhenAlreadyInitialized() throws Exception {
        McpExportConfig a = new McpExportConfig();
        a.setProperties(new HashMap<>());
        a.setRequired(Arrays.asList("B"));
        a.setMethods(Arrays.asList("A"));
        McpExportConfig b = new McpExportConfig();
        Map<String, Object> props = new HashMap<>();
        props.put("x", 1);
        b.setProperties(props);
        b.setRequired(Arrays.asList("r1"));
        b.setMethods(Arrays.asList("get"));
        a.merge(b);
        Assert.assertEquals(1, a.getProperties().get("x"));
        Assert.assertEquals("B", a.getRequired().getFirst());
        Assert.assertEquals("r1", a.getRequired().getLast());
        Assert.assertEquals("A", a.getMethods().getFirst());
        Assert.assertEquals("get", a.getMethods().getLast());
    }

    @Test
    public void testMergeMimeTypeDefaultsFromOtherGetter() throws Exception {
        McpExportConfig a = new McpExportConfig();
        a.setMimeType("");
        McpExportConfig b = new McpExportConfig();
        b.setMimeType(null);
        a.merge(b);
        Assert.assertEquals("text/plain", a.getMimeType());
    }
}
