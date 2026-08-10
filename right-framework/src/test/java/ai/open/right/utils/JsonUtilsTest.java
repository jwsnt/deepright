package ai.open.right.utils;

import org.junit.Assert;
import org.junit.Test;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.util.HashMap;
import java.util.Map;

public class JsonUtilsTest {

    @Test
    public void testWrite() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        String target = JsonUtils.write(test);
        Assert.assertEquals("{\"A\":\"B\",\"C\":{\"D\":\"E\"}}", target);
    }

    @Test
    public void testWriteSingleQuote() throws Exception {
        Assert.assertEquals("OK", JsonUtils.read("{'uri':'OK'}", Map.class).get("uri"));
    }

    @Test
    public void testWriteOutputStream() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        JsonUtils.write(outputStream, test);
        Assert.assertEquals("{\"A\":\"B\",\"C\":{\"D\":\"E\"}}", new String(outputStream.toByteArray()));
    }

    @Test
    public void testReadBytes() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        String target = JsonUtils.write(test);
        Map<String, Object> json = JsonUtils.read(target.getBytes(), Map.class);
        Assert.assertEquals(test.toString(), json.toString());
    }

    @Test
    public void testReadString() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        String target = JsonUtils.write(test);
        Map<String, Object> json = JsonUtils.read(target, Map.class);
        Assert.assertEquals(test.toString(), json.toString());
    }

    @Test
    public void testReadStream() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        String target = JsonUtils.write(test);
        Map<String, Object> json = JsonUtils.read(new ByteArrayInputStream(target.getBytes()), Map.class);
        Assert.assertEquals(test.toString(), json.toString());
    }

    @Test
    public void testInstance() throws Exception {
        Assert.assertEquals(JsonUtils.instance(), JsonUtils.MAPPER);
    }

    @Test
    public void testTransfer() throws Exception {
        Map<String, Object> test = new HashMap<String, Object>();
        test.put("A", "B");
        Map<String, Object> innr = new HashMap<String, Object>();
        innr.put("D", "E");
        test.put("C", innr);
        Map<String, Object> dest = JsonUtils.transfer(test, Map.class);
        Assert.assertEquals(dest.toString(), "{A=B, C={D=E}}");
    }

    @Test
    public void testCleanForJson() throws Exception {
        Assert.assertEquals("HELLO", JsonUtils.clean("```json\r\nHELLO\r\n```").trim());
    }

    @Test
    public void testCleanForJavaScript() throws Exception {
        Assert.assertEquals("HELLO", JsonUtils.clean("```javascript\r\nHELLO\r\n```").trim());
    }

    @Test
    public void testWriteString() throws Exception {
        String target = JsonUtils.write("HELLO");
        Assert.assertEquals("HELLO", target);
    }

    @Test
    public void testWriteStringOnOutputStream() throws Exception {
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        JsonUtils.write(outputStream, "HELLO");
        Assert.assertEquals("HELLO", new String(outputStream.toByteArray()));
    }

    @Test
    public void testExtract1() throws Exception {
        String json = "我理解你希望我比较两个模型对“世界杯”这个输入的输出结果，并给出详细的评分和理由。然而，你没有提供“回答1”和“回答2”的具体内容。为了进行有效的评估，我需要知道这两个模型实际上输出了什么。\n" +
                "\n" +
                "请提供输出A和输出B的具体内容，我才能按照你提供的评估标准和格式进行评估。\n" +
                "\n" +
                "**假设我拿到了以下两个模型的输出：**\n" +
                "\n" +
                "* **输出A:**  \"世界杯是全球最高荣誉的足球赛事，每四年举办一次，吸引了来自世界各地的顶级球队参与角逐。 它不仅是一场体育盛宴，也代表着文化交流和国家荣誉感。\"\n" +
                "* **输出B:**  \"世界杯，也称国际足联世界杯，是世界上最受欢迎的足球比赛。它由国际足球联合会（FIFA）主办，每四年举行一次。除了比赛，它也是一个巨大的商业活动，吸引了赞助商和游客。\"\n" +
                "\n" +
                "**那么，以下是一个基于假设的评估结果：**\n" +
                "\n" +
                "```json\n" +
                "{\n" +
                "    \"winner\": \"A\",\n" +
                "    \"scores\": {\n" +
                "        \"A\": {\n" +
                "            \"totalScore\": 88,\n" +
                "            \"reasoning\": \"输出A提供了关于世界杯更完整和更具吸引力的描述。它强调了赛事的荣誉感、文化交流，使其不仅仅是体育赛事，更具人文价值。语言流畅，更富感情色彩，更容易引起共鸣。 唯一的不足是缺乏具体的主办方信息。\"\n" +
                "        },\n" +
                "        \"B\": {\n" +
                "            \"totalScore\": 82,\n" +
                "            \"reasoning\": \"输出B简洁明了地解释了世界杯的名称、主办方和举办周期。它也提到了世界杯的商业价值，这增加了描述的全面性。但相较于A，略显平淡，缺乏情感连接，且信息密度略低。\"\n" +
                "        }\n" +
                "    },\n" +
                "    \"comparison\": \"A和B都提供了关于世界杯的基本信息。A更注重情感和荣誉感，描述更生动；B则更注重客观事实和商业价值。 A在吸引力方面胜过B，但B在提供主办方信息方面优于A。 A在整体上更完整和吸引人，因此胜出。\",\n" +
                "    \"confidence\": 0.90\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "**评分维度和权重（假设）：**\n" +
                "\n" +
                "* **信息完整性 (30%)**:  模型是否提供了关于主题的关键信息。\n" +
                "* **准确性 (20%)**:  模型提供的信息是否准确无误。\n" +
                "* **语言质量 (25%)**:  模型使用的语言是否流畅、自然、易于理解。\n" +
                "* **相关性 (15%)**:  模型输出的内容是否与用户输入密切相关。\n" +
                "* **洞察力/附加价值 (10%)**: 模型是否提供了一些额外的洞察力或价值，例如独特的视角或更深入的解释。\n" +
                "\n" +
                "请提供实际的输出A和输出B，我将根据这些内容进行更准确的评估。\n";
        String expect = "{\n" +
                "    \"winner\": \"A\",\n" +
                "    \"scores\": {\n" +
                "        \"A\": {\n" +
                "            \"totalScore\": 88,\n" +
                "            \"reasoning\": \"输出A提供了关于世界杯更完整和更具吸引力的描述。它强调了赛事的荣誉感、文化交流，使其不仅仅是体育赛事，更具人文价值。语言流畅，更富感情色彩，更容易引起共鸣。 唯一的不足是缺乏具体的主办方信息。\"\n" +
                "        },\n" +
                "        \"B\": {\n" +
                "            \"totalScore\": 82,\n" +
                "            \"reasoning\": \"输出B简洁明了地解释了世界杯的名称、主办方和举办周期。它也提到了世界杯的商业价值，这增加了描述的全面性。但相较于A，略显平淡，缺乏情感连接，且信息密度略低。\"\n" +
                "        }\n" +
                "    },\n" +
                "    \"comparison\": \"A和B都提供了关于世界杯的基本信息。A更注重情感和荣誉感，描述更生动；B则更注重客观事实和商业价值。 A在吸引力方面胜过B，但B在提供主办方信息方面优于A。 A在整体上更完整和吸引人，因此胜出。\",\n" +
                "    \"confidence\": 0.90\n" +
                "}";
        Assert.assertEquals(expect, JsonUtils.extract(json));
    }

    @Test
    public void testExtract2() throws Exception {
        String json = "我理解你希望我比较两个模型对“世界杯”这个输入的输出结果，并给出详细的评分和理由。然而，你没有提供“回答1”和“回答2”的具体内容。为了进行有效的评估，我需要知道这两个模型实际上输出了什么。\n" +
                "\n" +
                "请提供输出A和输出B的具体内容，我才能按照你提供的评估标准和格式进行评估。\n" +
                "\n" +
                "**假设我拿到了以下两个模型的输出：**\n" +
                "\n" +
                "* **输出A:**  \"世界杯是全球最高荣誉的足球赛事，每四年举办一次，吸引了来自世界各地的顶级球队参与角逐。 它不仅是一场体育盛宴，也代表着文化交流和国家荣誉感。\"\n" +
                "* **输出B:**  \"世界杯，也称国际足联世界杯，是世界上最受欢迎的足球比赛。它由国际足球联合会（FIFA）主办，每四年举行一次。除了比赛，它也是一个巨大的商业活动，吸引了赞助商和游客。\"\n" +
                "\n" +
                "**那么，以下是一个基于假设的评估结果：**\n" +
                "**评分维度和权重（假设）：**\n" +
                "\n" +
                "* **信息完整性 (30%)**:  模型是否提供了关于主题的关键信息。\n" +
                "* **准确性 (20%)**:  模型提供的信息是否准确无误。\n" +
                "* **语言质量 (25%)**:  模型使用的语言是否流畅、自然、易于理解。\n" +
                "* **相关性 (15%)**:  模型输出的内容是否与用户输入密切相关。\n" +
                "* **洞察力/附加价值 (10%)**: 模型是否提供了一些额外的洞察力或价值，例如独特的视角或更深入的解释。\n" +
                "\n" +
                "请提供实际的输出A和输出B，我将根据这些内容进行更准确的评估。\n";
        Assert.assertEquals(json, JsonUtils.extract(json));
    }

    @Test
    public void testExtract3() throws Exception {
        String json = "我理解你希望我比较两个模型对“世界杯”这个输入的输出结果，并给出详细的评分和理由。然而，你没有提供“回答1”和“回答2”的具体内容。为了进行有效的评估，我需要知道这两个模型实际上输出了什么。\n" +
                "\n" +
                "请提供输出A和输出B的具体内容，我才能按照你提供的评估标准和格式进行评估。\n" +
                "\n" +
                "**假设我拿到了以下两个模型的输出：**\n" +
                "\n" +
                "* **输出A:**  \"世界杯是全球最高荣誉的足球赛事，每四年举办一次，吸引了来自世界各地的顶级球队参与角逐。 它不仅是一场体育盛宴，也代表着文化交流和国家荣誉感。\"\n" +
                "* **输出B:**  \"世界杯，也称国际足联世界杯，是世界上最受欢迎的足球比赛。它由国际足球联合会（FIFA）主办，每四年举行一次.除了比赛，它也是一个巨大的商业活动，吸引了赞助商和游客。\"\n" +
                "\n" +
                "**那么，以下是一个基于假设的评估结果：**\n" +
                "\n" +
                "```json\n" +
                "{\n" +
                "    \"winner\": \"A\",\n" +
                "    \"scores\": {\n" +
                "        \"A\": {\n" +
                "            \"totalScore\": 88,\n" +
                "            \"reasoning\": \"输出A提供了关于世界杯更完整和更具吸引力的描述。它强调了赛事的荣誉感、文化交流，使其不仅仅是体育赛事，更具人文价值。语言流畅，更富感情色彩，更容易引起共鸣。 唯一的不足是缺乏具体的主办方信息。\"\n" +
                "        },\n" +
                "        \"B\": {\n" +
                "            \"totalScore\": 82,\n" +
                "            \"reasoning\": \"输出B简洁明了地解释了世界杯的名称、主办方和举办周期。它也提到了世界杯的商业价值，这增加了描述的全面性。但相较于A，略显平淡，缺乏情感连接，且信息密度略低。\"\n" +
                "        }\n" +
                "    },\n" +
                "    \"comparison\": \"A和B都提供了关于世界杯的基本信息。A更注重情感和荣誉感，描述更生动；B则更注重客观事实和商业价值。 A在吸引力方面胜过B，但B在提供主办方信息方面优于A。 A在整体上更完整和吸引人，因此胜出。\",\n" +
                "    \"confidence\": 0.90\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "**评分维度和权重（假设）：**\n" +
                "\n" +
                "* **信息完整性 (30%)**:  模型是否提供了关于主题的关键信息。\n" +
                "* **准确性 (20%)**:  模型提供的信息是否准确无误。\n" +
                "* **语言质量 (25%)**:  模型使用的语言是否流畅、自然、易于理解。\n" +
                "* **相关性 (15%)**:  模型输出的内容是否与用户输入密切相关。\n" +
                "* **洞察力/附加价值 (10%)**: 模型是否提供了一些额外的洞察力或价值，例如独特的视角或更深入的解释。\n" +
                "\n" +
                "```json\n" +
                "{\n" +
                "    \"winner2\": \"A\",\n" +
                "    \"scores2\": {\n" +
                "        \"A\": {\n" +
                "            \"totalScore\": 88,\n" +
                "            \"reasoning\": \"输出A提供了关于世界杯更完整和更具吸引力的描述。它强调了赛事的荣誉感、文化交流，使其不仅仅是体育赛事，更具人文价值。语言流畅，更富感情色彩，更容易引起共鸣。 唯一的不足是缺乏具体的主办方信息。\"\n" +
                "        },\n" +
                "        \"B\": {\n" +
                "            \"totalScore\": 82,\n" +
                "            \"reasoning\": \"输出B简洁明了地解释了世界杯的名称、主办方和举办周期。它也提到了世界杯的商业价值，这增加了描述的全面性。但相较于A，略显平淡，缺乏情感连接，且信息密度略低。\"\n" +
                "        }\n" +
                "    },\n" +
                "    \"comparison2\": \"A和B都提供了关于世界杯的基本信息。A更注重情感和荣誉感，描述更生动；B则更注重客观事实和商业价值。 A在吸引力方面胜过B，但B在提供主办方信息方面优于A。 A在整体上更完整和吸引人，因此胜出。\",\n" +
                "    \"confidence2\": 0.90\n" +
                "}\n" +
                "```\n" +
                "请提供实际的输出A和输出B，我将根据 these 内容进行更准确的评估。\n";
        String expect = "{\n" +
                "    \"winner\": \"A\",\n" +
                "    \"scores\": {\n" +
                "        \"A\": {\n" +
                "            \"totalScore\": 88,\n" +
                "            \"reasoning\": \"输出A提供了关于世界杯更完整和更具吸引力的描述。它强调了赛事的荣誉感、文化交流，使其不仅仅是体育赛事，更具人文价值。语言流畅，更富感情色彩，更容易引起共鸣。 唯一的不足是缺乏具体的主办方信息。\"\n" +
                "        },\n" +
                "        \"B\": {\n" +
                "            \"totalScore\": 82,\n" +
                "            \"reasoning\": \"输出B简洁明了地解释了世界杯的名称、主办方和举办周期。它也提到了世界杯的商业价值，这增加了描述的全面性。但相较于A，略显平淡，缺乏情感连接，且信息密度略低。\"\n" +
                "        }\n" +
                "    },\n" +
                "    \"comparison\": \"A和B都提供了关于世界杯的基本信息。A更注重情感和荣誉感，描述更生动；B则更注重客观事实和商业价值。 A在吸引力方面胜过B，但B在提供主办方信息方面优于A。 A在整体上更完整和吸引人，因此胜出。\",\n" +
                "    \"confidence\": 0.90\n" +
                "}";
        Assert.assertEquals(expect, JsonUtils.extract(json));
    }

    @Test
    public void testReadAsString() throws Exception {
        Assert.assertEquals("ABC", JsonUtils.read("ABC".getBytes(), String.class));
        Assert.assertEquals("ABC", JsonUtils.read(new ByteArrayInputStream("ABC".getBytes()), String.class));
        Assert.assertEquals("ABC", JsonUtils.read("ABC", String.class));
    }


    @Test
    public void testLikeMap() throws Exception {
        Assert.assertTrue(JsonUtils.map("{}"));
        Assert.assertTrue(JsonUtils.map("{\"a\":\"b\"}"));
        Assert.assertFalse(JsonUtils.map("{\"a\":\"b\"]"));
    }

    @Test
    public void testLikeArray() throws Exception {
        Assert.assertFalse(JsonUtils.array(null));
        Assert.assertTrue(JsonUtils.array("[]"));
        Assert.assertTrue(JsonUtils.array("[\"a\":\"b\"]"));
        Assert.assertFalse(JsonUtils.array("{\"a\":\"b\"]"));
    }

    @Test
    public void testCleanEmpty() throws Exception {
        Assert.assertEquals("", JsonUtils.clean(""));
    }

    @Test
    public void testCleanOnlySuffix() throws Exception {
        Assert.assertEquals("", JsonUtils.clean("```"));
    }

    @Test
    public void testExtractNested() throws Exception {
        String nested = "```json\n{\"a\": \"```json inner ```\"}\n```";
        // The pattern is "```json\\s*([\\s\\S]*?)```"
        // It will match from the first ```json to the first ``` after it.
        Assert.assertEquals("{\"a\": \"", JsonUtils.extract(nested));
    }

    @Test
    public void testLikeMalformed() throws Exception {
        Assert.assertFalse(JsonUtils.like("{\"a\":\"b\""));
        Assert.assertFalse(JsonUtils.like("[1, 2"));
    }

    @Test
    public void testMapMalformed() throws Exception {
        Assert.assertFalse(JsonUtils.map("{\"a\":\"b\""));
    }

    @Test
    public void testCleanNoLanguage() throws Exception {
        Assert.assertEquals("HELLO", JsonUtils.clean("```\nHELLO\n```"));
    }

    @org.junit.jupiter.api.Test
    public void testCleanIncompleteMarkdown() throws Exception {
        String input = "```json {\"key\": \"value\"}";
        org.junit.jupiter.api.Assertions.assertEquals(input, JsonUtils.clean(input));
    }

    @org.junit.jupiter.api.Test
    public void testLikeAndMapInvalidJson() throws Exception {
        String invalid = "{ this is not json }";
        org.junit.jupiter.api.Assertions.assertTrue(JsonUtils.like(invalid));
        org.junit.jupiter.api.Assertions.assertTrue(JsonUtils.map(invalid));
    }

    @org.junit.jupiter.api.Test
    public void testReadUnknownProperties() throws Exception {
        String json = "{\"name\": \"test\", \"unknown_field\": 123}";
        java.util.Map bean = JsonUtils.read(json, java.util.Map.class);
        org.junit.jupiter.api.Assertions.assertNotNull(bean);
        org.junit.jupiter.api.Assertions.assertEquals("test", bean.get("name"));
    }

    @org.junit.jupiter.api.Test
    public void testReadNull() throws Exception {
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.read((String) null, java.util.Map.class));
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.read((byte[]) null, java.util.Map.class));
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.read((java.io.InputStream) null, java.util.Map.class));
    }

    @org.junit.jupiter.api.Test
    public void testWriteNull() throws Exception {
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.write(null));
    }

    @org.junit.jupiter.api.Test
    public void testExtractNullAndEmpty() throws Exception {
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.extract(null));
        org.junit.jupiter.api.Assertions.assertEquals("", JsonUtils.extract(""));
    }

    @org.junit.jupiter.api.Test
    public void testCleanNull() throws Exception {
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.clean(null));
    }

    @org.junit.jupiter.api.Test
    public void testTransferNull() throws Exception {
        org.junit.jupiter.api.Assertions.assertNull(JsonUtils.transfer(null, java.util.Map.class));
    }

    @org.junit.jupiter.api.Test
    public void testLikeAndMapNullAndEmpty() throws Exception {
        org.junit.jupiter.api.Assertions.assertFalse(JsonUtils.like(null));
        org.junit.jupiter.api.Assertions.assertFalse(JsonUtils.like(""));
        org.junit.jupiter.api.Assertions.assertFalse(JsonUtils.map(null));
        org.junit.jupiter.api.Assertions.assertFalse(JsonUtils.map(""));
    }

    @org.junit.jupiter.api.Test
    public void testExtractNoMatch() throws Exception {
        String input = "no json here";
        org.junit.jupiter.api.Assertions.assertEquals(input, JsonUtils.extract(input));
    }

    @org.junit.jupiter.api.Test
    public void testCleanOnlyPrefix() throws Exception {
        org.junit.jupiter.api.Assertions.assertEquals("", JsonUtils.clean("```json"));
    }

    @org.junit.jupiter.api.Test
    public void testReadInvalidJsonException() throws Exception {
        org.junit.jupiter.api.Assertions.assertThrows(Exception.class, () -> {
            JsonUtils.read("{invalid}", Map.class);
        });
    }

    @org.junit.jupiter.api.Test
    public void testWriteComplexObject() throws Exception {
        Map<String, Object> complex = new HashMap<>();
        complex.put("key", "value");
        complex.put("list", java.util.Arrays.asList(1, 2, 3));
        String result = JsonUtils.write(complex);
        org.junit.jupiter.api.Assertions.assertTrue(result.contains("\"key\":\"value\""));
        org.junit.jupiter.api.Assertions.assertTrue(result.contains("\"list\":[1,2,3]"));
    }

    @Test
    public void testJson() throws Exception {
        Assert.assertTrue(JsonUtils.like("\n" +
                "\n" +
                "```JSON\n" +
                "{\"needs\":\"安全需求\",\"digest\":\"用户询问东北景点天气\",\"important\":\"0\",\"why_do_this\":\"用户仅询问天气信息，助手回复查询中，属于回答已有问题无新知识获取，符合类型0\"}\n" +
                "```"));
    }
}

