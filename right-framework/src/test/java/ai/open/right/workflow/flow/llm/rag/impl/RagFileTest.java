package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class RagFileTest {

    @Test
    public void test() throws Exception {
        RagFile ragFile = new RagFile();
        ragFile.setPlaceholderResolver(new PlaceholderResolver() {

            @Override
            public String replace(String input) throws Exception {
                return input;
            }
        });
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setFile("classpath:A2A.json");
        ragFile.setResourceService(ObjectBuilder.buildResourceService());
        ragFile.rag(ragConfig, ragData);
        String expected = "UNKNOWN [\n" +
                "  {\n" +
                "    \"name\": \"video-generator@a2a\",\n" +
                "    \"description\": \"Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\n" +
                "    \"preferredTransport\": \"JSONRPC\",\n" +
                "    \"version\": \"1.2.0\",\n" +
                "    \"documentationUrl\": \"https://docs.examplegeoservices.com/georoute-agent/api\",\n" +
                "    \"capabilities\": {\n" +
                "      \"streaming\": true,\n" +
                "      \"pushNotifications\": true,\n" +
                "      \"stateTransitionHistory\": false\n" +
                "    },\n" +
                "    \"defaultInputModes\": [\n" +
                "      \"application/json\",\n" +
                "      \"text/plain\"\n" +
                "    ],\n" +
                "    \"defaultOutputModes\": [\n" +
                "      \"application/json\",\n" +
                "      \"image/png\"\n" +
                "    ],\n" +
                "    \"skills\": [\n" +
                "      {\n" +
                "        \"id\": \"route-optimizer-traffic\",\n" +
                "        \"name\": \"Traffic-Aware Route Optimizer\",\n" +
                "        \"description\": \"Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\n" +
                "        \"tags\": [\n" +
                "          \"maps\",\n" +
                "          \"routing\",\n" +
                "          \"navigation\",\n" +
                "          \"directions\",\n" +
                "          \"traffic\"\n" +
                "        ],\n" +
                "        \"examples\": [\n" +
                "          \"Plan a route from '1600 Amphitheatre Parkway, Mountain View, CA' to 'San Francisco International Airport' avoiding tolls.\",\n" +
                "          \"{\\\"origin\\\": {\\\"lat\\\": 37.422, \\\"lng\\\": -122.084}, \\\"destination\\\": {\\\"lat\\\": 37.7749, \\\"lng\\\": -122.4194}, \\\"preferences\\\": [\\\"avoid_ferries\\\"]}\"\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  },\n" +
                "  {\n" +
                "    \"name\": \"video-generator@a2a_stream\",\n" +
                "    \"description\": \"Stream Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\n" +
                "    \"preferredTransport\": \"JSONRPC\",\n" +
                "    \"version\": \"1.2.0\",\n" +
                "    \"documentationUrl\": \"https://docs.examplegeoservices.com/georoute-agent-stream/api\",\n" +
                "    \"capabilities\": {\n" +
                "      \"streaming\": true,\n" +
                "      \"pushNotifications\": true,\n" +
                "      \"stateTransitionHistory\": true\n" +
                "    },\n" +
                "    \"defaultInputModes\": [\n" +
                "      \"application/json\"\n" +
                "    ],\n" +
                "    \"defaultOutputModes\": [\n" +
                "      \"image/png\"\n" +
                "    ],\n" +
                "    \"skills\": [\n" +
                "      {\n" +
                "        \"id\": \"Stream  route-optimizer-traffic\",\n" +
                "        \"name\": \"Stream  Traffic-Aware Route Optimizer\",\n" +
                "        \"description\": \"Stream  Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\n" +
                "        \"tags\": [\n" +
                "          \"maps\",\n" +
                "          \"routing\"\n" +
                "        ],\n" +
                "        \"examples\": [\n" +
                "          \"Stream plan a route from '1600 Amphitheatre Parkway, Mountain View, CA' to 'San Francisco International Airport' avoiding tolls.\",\n" +
                "          \"{\\\"origin\\\": {\\\"lat\\\": 37.422, \\\"lng\\\": -122.084}, \\\"destination\\\": {\\\"lat\\\": 37.7749, \\\"lng\\\": -122.4194}, \\\"preferences\\\": [\\\"avoid_ferries\\\"]}\"\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  }\n" +
                "]";
        Assert.assertNotNull(ragFile.getResourceService());
        Assert.assertEquals(expected, ragData.getPrompt());
    }

    @Test
    public void testNotAllowed() throws Exception {
        RagFile ragFile = new RagFile() {
            protected Boolean allowed(RagConfig ragConfig, RagData ragData) throws Exception {
                return false;
            }
        };
        ragFile.setPlaceholderResolver(new PlaceholderResolver() {

            @Override
            public String replace(String input) throws Exception {
                return input;
            }
        });
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setFile("classpath:A2A.json");
        Assert.assertEquals(RagFuture.NOTHING, ragFile.rag(ragConfig, ragData));
    }

    @Test
    public void testCache() throws Exception {
        RagFile ragFile = new RagFile();
        ragFile.setResourceService(ObjectBuilder.buildResourceService());
        ragFile.setPlaceholderResolver(new PlaceholderResolver() {

            @Override
            public String replace(String input) throws Exception {
                return input;
            }
        });
        LLMQuery query = ObjectBuilder.buildLLMQuery();
        RagData ragData = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig = new RagConfig();
        ragConfig.setReplace("#key");
        ragConfig.setFile("classpath:A2A.json");
        ragFile.rag(ragConfig, ragData);
        String expected = "UNKNOWN [\n" +
                "  {\n" +
                "    \"name\": \"video-generator@a2a\",\n" +
                "    \"description\": \"Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\n" +
                "    \"preferredTransport\": \"JSONRPC\",\n" +
                "    \"version\": \"1.2.0\",\n" +
                "    \"documentationUrl\": \"https://docs.examplegeoservices.com/georoute-agent/api\",\n" +
                "    \"capabilities\": {\n" +
                "      \"streaming\": true,\n" +
                "      \"pushNotifications\": true,\n" +
                "      \"stateTransitionHistory\": false\n" +
                "    },\n" +
                "    \"defaultInputModes\": [\n" +
                "      \"application/json\",\n" +
                "      \"text/plain\"\n" +
                "    ],\n" +
                "    \"defaultOutputModes\": [\n" +
                "      \"application/json\",\n" +
                "      \"image/png\"\n" +
                "    ],\n" +
                "    \"skills\": [\n" +
                "      {\n" +
                "        \"id\": \"route-optimizer-traffic\",\n" +
                "        \"name\": \"Traffic-Aware Route Optimizer\",\n" +
                "        \"description\": \"Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\n" +
                "        \"tags\": [\n" +
                "          \"maps\",\n" +
                "          \"routing\",\n" +
                "          \"navigation\",\n" +
                "          \"directions\",\n" +
                "          \"traffic\"\n" +
                "        ],\n" +
                "        \"examples\": [\n" +
                "          \"Plan a route from '1600 Amphitheatre Parkway, Mountain View, CA' to 'San Francisco International Airport' avoiding tolls.\",\n" +
                "          \"{\\\"origin\\\": {\\\"lat\\\": 37.422, \\\"lng\\\": -122.084}, \\\"destination\\\": {\\\"lat\\\": 37.7749, \\\"lng\\\": -122.4194}, \\\"preferences\\\": [\\\"avoid_ferries\\\"]}\"\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  },\n" +
                "  {\n" +
                "    \"name\": \"video-generator@a2a_stream\",\n" +
                "    \"description\": \"Stream Provides advanced route planning, traffic analysis, and custom map generation services. This agent can calculate optimal routes, estimate travel times considering real-time traffic, and create personalized maps with points of interest.\",\n" +
                "    \"preferredTransport\": \"JSONRPC\",\n" +
                "    \"version\": \"1.2.0\",\n" +
                "    \"documentationUrl\": \"https://docs.examplegeoservices.com/georoute-agent-stream/api\",\n" +
                "    \"capabilities\": {\n" +
                "      \"streaming\": true,\n" +
                "      \"pushNotifications\": true,\n" +
                "      \"stateTransitionHistory\": true\n" +
                "    },\n" +
                "    \"defaultInputModes\": [\n" +
                "      \"application/json\"\n" +
                "    ],\n" +
                "    \"defaultOutputModes\": [\n" +
                "      \"image/png\"\n" +
                "    ],\n" +
                "    \"skills\": [\n" +
                "      {\n" +
                "        \"id\": \"Stream  route-optimizer-traffic\",\n" +
                "        \"name\": \"Stream  Traffic-Aware Route Optimizer\",\n" +
                "        \"description\": \"Stream  Calculates the optimal driving route between two or more locations, taking into account real-time traffic conditions, road closures, and user preferences (e.g., avoid tolls, prefer highways).\",\n" +
                "        \"tags\": [\n" +
                "          \"maps\",\n" +
                "          \"routing\"\n" +
                "        ],\n" +
                "        \"examples\": [\n" +
                "          \"Stream plan a route from '1600 Amphitheatre Parkway, Mountain View, CA' to 'San Francisco International Airport' avoiding tolls.\",\n" +
                "          \"{\\\"origin\\\": {\\\"lat\\\": 37.422, \\\"lng\\\": -122.084}, \\\"destination\\\": {\\\"lat\\\": 37.7749, \\\"lng\\\": -122.4194}, \\\"preferences\\\": [\\\"avoid_ferries\\\"]}\"\n" +
                "        ]\n" +
                "      }\n" +
                "    ]\n" +
                "  }\n" +
                "]";
        Assert.assertEquals(expected, ragData.getPrompt());
        RagData ragData2 = RagData.builder()
                .config(new LLMConfig())
                .prompt("UNKNOWN #key")
                .query(query)
                .build();
        RagConfig ragConfig2 = new RagConfig();
        ragConfig2.setReplace("#key");
        ragConfig2.setFile("classpath:CustomerTools.json");
        ragFile.rag(ragConfig2, ragData2);
        expected = "UNKNOWN {\n" +
                "  \"_fun_customer\": [\n" +
                "    {\n" +
                "      \"description\": \"获取天气\",\n" +
                "      \"name\": \"workflow2\",\n" +
                "      \"properties\": {\n" +
                "        \"city\": {\n" +
                "          \"type\": \"string\",\n" +
                "          \"description\": \"城市名称,可以是多个\"\n" +
                "        },\n" +
                "        \"description\": {\n" +
                "          \"type\": \"string\",\n" +
                "          \"description\": \"时间范围，比如最近几个月\"\n" +
                "        }\n" +
                "      },\n" +
                "      \"required\": [\n" +
                "        \"city\"\n" +
                "      ]\n" +
                "    },\n" +
                "    {\n" +
                "      \"description\": \"获取用户所在地区\",\n" +
                "      \"name\": \"workflow3\",\n" +
                "      \"properties\": {\n" +
                "        \"location\": {\n" +
                "          \"type\": \"string\",\n" +
                "          \"description\": \"比如华东或华北\"\n" +
                "        }\n" +
                "      },\n" +
                "      \"required\": [\n" +
                "        \"location\"\n" +
                "      ]\n" +
                "    }\n" +
                "  ]\n" +
                "}";
        Assert.assertEquals(expected, ragData2.getPrompt());
    }

    @Test
    public void testInit() throws Exception {
        NotifierServiceImpl notifierManager = ObjectBuilder.buildActualNotifierManagerWithNothing();
        PlaceholderResolver placeholderResolver = EasyMock.createMock(PlaceholderResolver.class);
        EasyMock.replay(placeholderResolver);
        RagFile.InitConfig service = new RagFile.InitConfig();
        service.setNotifierService(notifierManager);
        service.setPlaceholderResolver(placeholderResolver);
        service.setTimeout4Condition(10086);
        RagFile empty = service.ragFile();
        Assert.assertEquals(notifierManager, empty.getNotifierService());
        Assert.assertEquals(placeholderResolver, empty.getPlaceholderResolver());
        Assert.assertEquals(Integer.valueOf(10086), empty.getTimeout4Condition());
        EasyMock.verify(placeholderResolver);
    }
}
