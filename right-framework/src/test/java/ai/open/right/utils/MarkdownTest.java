package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

/**
 * {@link Markdown#object(String)}：JSON Schema 风格对象 → Markdown 表格。
 * {@link Markdown#array(String)}：JSON Schema 数组（取 {@code items} 子对象）→ 同上。
 */
public class MarkdownTest {

    @Test
    public void md_emitsHeaderAndRows() throws Exception {
        String json = "{\"properties\":{\"foo\":{\"type\":\"string\",\"description\":\"bar\"}},\"required\":[\"foo\"]}";
        String out = Markdown.object(json);
        Assert.assertTrue(out.contains("|Field|Type|Required|Description|"));
        Assert.assertTrue(out.contains("|:---|:---|:---|:---|"));
        Assert.assertTrue(out.contains("|**foo**|`string`|true|bar|"));
    }

    @Test
    public void md_requiredFalseWhenNotInRequiredArray() throws Exception {
        String json = "{\"properties\":{\"a\":{\"type\":\"integer\",\"description\":\"n\"}},\"required\":[]}";
        String out = Markdown.object(json);
        Assert.assertTrue(out.contains("|**a**|`integer`|false|n|"));
    }

    @Test
    public void md_missingRequiredArray_treatsAllOptional() throws Exception {
        String json = "{\"properties\":{\"x\":{\"type\":\"boolean\",\"description\":\"d\"}}}";
        String out = Markdown.object(json);
        Assert.assertTrue(out.contains("|**x**|`boolean`|false|d|"));
    }

    @Test
    public void md_missingTypeAndDescription_useDash() throws Exception {
        String json = "{\"properties\":{\"bare\":{}},\"required\":[]}";
        String out = Markdown.object(json);
        Assert.assertTrue(out.contains("|**bare**|`-`|false|-|"));
    }

    @Test
    public void md_emptyProperties_onlyTableHeader() throws Exception {
        String json = "{\"properties\":{},\"required\":[]}";
        String out = Markdown.object(json);
        Assert.assertTrue(out.startsWith("|Field|Type|Required|Description|" + System.lineSeparator()));
        Assert.assertTrue(out.contains("|:---|:---|:---|:---|"));
        Assert.assertFalse(out.contains("|**"));
    }

    @Test
    public void md_multipleFields_orderFollowsJsonFieldOrder() throws Exception {
        String json = "{\"properties\":{\"z\":{\"type\":\"string\"},\"a\":{\"type\":\"number\"}},\"required\":[\"a\"]}";
        String out = Markdown.object(json);
        int z = out.indexOf("|**z**|");
        int a = out.indexOf("|**a**|");
        Assert.assertTrue(z > 0 && a > z);
    }

    @Test
    public void array_extractsItems_delegatesToObject() throws Exception {
        String itemsOnly = "{\"properties\":{\"foo\":{\"type\":\"string\",\"description\":\"bar\"}},\"required\":[\"foo\"]}";
        String arrayJson = "{\"type\":\"array\",\"items\":{\"properties\":{\"foo\":{\"type\":\"string\",\"description\":\"bar\"}},\"required\":[\"foo\"]}}";
        Assert.assertEquals(Markdown.object(itemsOnly), Markdown.array(arrayJson));
    }

    @Test
    public void array_nestedObjectSchema_sameAsObject() throws Exception {
        String itemsOnly = "{\"type\":\"object\",\"id\":\"urn:x\",\"properties\":{\"a\":{\"type\":\"integer\",\"description\":\"n\"}},\"required\":[]}";
        String arrayJson = "{\"type\":\"array\",\"items\":{\"type\":\"object\",\"id\":\"urn:x\",\"properties\":{\"a\":{\"type\":\"integer\",\"description\":\"n\"}},\"required\":[]}}";
        Assert.assertEquals(Markdown.object(itemsOnly), Markdown.array(arrayJson));
    }

    @Test
    public void array_multipleFields_orderMatchesObject() throws Exception {
        String itemsOnly = "{\"properties\":{\"z\":{\"type\":\"string\"},\"a\":{\"type\":\"number\"}},\"required\":[\"a\"]}";
        String arrayJson = "{\"items\":{\"properties\":{\"z\":{\"type\":\"string\"},\"a\":{\"type\":\"number\"}},\"required\":[\"a\"]}}";
        String fromObject = Markdown.object(itemsOnly);
        String fromArray = Markdown.array(arrayJson);
        Assert.assertEquals(fromObject, fromArray);
        int z = fromArray.indexOf("|**z**|");
        int a = fromArray.indexOf("|**a**|");
        Assert.assertTrue(z > 0 && a > z);
    }
}
