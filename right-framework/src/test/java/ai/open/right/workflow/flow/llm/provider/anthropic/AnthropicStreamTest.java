package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.provider.ProviderReader;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.signal.SignalExecutor;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import org.apache.commons.io.FileUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.io.comparator.NameFileComparator;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.ResourceUtils;

import java.io.File;
import java.nio.charset.StandardCharsets;
import java.util.*;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
public class AnthropicStreamTest {

    @Test
    public void testCallback() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        c.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        c.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Anthropic_response_atonce.json").openStream(), StandardCharsets.UTF_8);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected Boolean atonce(String message) throws Exception {
                Assert.assertEquals(content, message);
                return true;
            }

            @Override
            protected void afterAtOnce() throws Exception {
            }
        };
        stream.callback(content);
    }

    @Test
    public void testCallbackWithDone() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        c.appendRequest(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        c.appendResponse(EasyMock.anyString());
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected Boolean atonce(String message) throws Exception {
                Assert.fail();
                return true;
            }

            @Override
            protected void afterAtOnce() throws Exception {
            }
        };
        stream.callback(ProviderReader.DONE);
    }

    @Test
    public void testAtOnce() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Anthropic_response_atonce.json").openStream(), StandardCharsets.UTF_8);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(1342), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
            }

            @Override
            public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                return List.of();
            }

            @Override
            public List<TokenData> readAll(Dimension dimension) throws Exception {
                return List.of();
            }

            @Override
            public TokenData read(Dimension dimension, String model) throws Exception {
                return null;
            }

            @Override
            public TokenData read(Dimension dimension) throws Exception {
                return null;
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        stream.atonce(content);
        Assert.assertEquals("首先，用户要求我扮演代码自动化CR助理，帮助检查提交的代码有没有质量问题。要求包括：\n" +
                "- 输出在2000字以内，并在结尾处输出`[FINISH]`。\n" +
                "- 首先输出有问题的代码，然后输出需要修改的问题。\n" +
                "- 不回答无关内容。\n" +
                "\n" +
                "用户提供的代码是Java代码，文件名为`MainApplication.java`，位于包`ai.open.right`中。这是一个Spring Boot应用程序的主类。\n" +
                "\n" +
                "我需要检查代码中的质量问题。Spring Boot应用程序的主类通常很简单，但可能有一些常见问题。我将回顾Spring Boot的最佳实践。\n" +
                "\n" +
                "代码内容：\n" +
                "```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "现在，分析可能的问题：\n" +
                "\n" +
                "1. **`@PropertySource` 注解**：在Spring Boot中，通常不需要显式使用`@PropertySource`来加载`application.properties`，因为Spring Boot会自动加载`src/main/resources`目录下的`application.properties`文件。除非有特殊原因，否则这可能是多余的。\n" +
                "\n" +
                "2. **`@EnableAsync` 注解**：使用了`proxyTargetClass = true`，这指定了使用CGLIB代理而不是Java接口代理。如果不需要CGLIB代理，可以省略或设置为`false`，以避免不必要的复杂性。通常，`@EnableAsync` 默认使用JDK动态代理，除非指定了`proxyTargetClass`。\n" +
                "\n" +
                "3. **`@SpringBootApplication` 注解**：`exclude` 参数排除了`DataSourceAutoConfiguration`，这可能是因为应用程序不需要数据库连接。但如果有配置错误，可能会导致问题。例如，如果其他配置类期望数据源，但没有正确配置，可能会抛出异常。\n" +
                "\n" +
                "4. **`main` 方法**：使用了`throws Exception`，这不是最佳实践。通常，应该捕获特定的异常或让Spring Boot处理异常。`throws Exception` 可能会掩盖具体的错误。\n" +
                "\n" +
                "5. **异常处理**：在`main`方法中，直接抛出异常可能不是理想的方式。Spring Boot应用程序通常会优雅地启动，如果发生错误，会记录日志并退出。最好移除`throws Exception`并让Spring Boot处理异常。\n" +
                "\n" +
                "6. **类名和包名**：类名`MainApplication`是合理的，包名`ai.open.right`看起来也不错。没有明显问题。\n" +
                "\n" +
                "7. **注释和文档**：代码中没有注释，但可能不需要，因为是主类。\n" +
                "\n" +
                "现在，根据要求，我需要先输出有问题的代码，然后输出需要修改的问题。\n" +
                "\n" +
                "输出格式：\n" +
                "- 首先，输出有问题的代码。用户已经提供了代码，所以我可以直接复制它。\n" +
                "- 然后，输出需要修改的问题列表。\n" +
                "\n" +
                "字数限制：2000字以内。代码本身很短，所以应该没问题。\n" +
                "\n" +
                "开始起草响应。\n" +
                "\n" +
                "**有问题的代码**：\n" +
                "复制提供的代码。\n" +
                "\n" +
                "**需要修改的问题**：\n" +
                "1. **移除多余的`@PropertySource`注解**：Spring Boot自动加载`application.properties`，除非有特殊需求，否则不需要显式声明。\n" +
                "2. **简化`@EnableAsync`注解**：如果不需要CGLIB代理，移除`proxyTargetClass = true`或设置为`false`。\n" +
                "3. **改进异常处理**：从`main`方法中移除`throws Exception`，并让Spring Boot处理启动异常。\n" +
                "4. **检查数据源排除**：确保排除`DataSourceAutoConfiguration`是必要的，并且不会导致其他问题。\n" +
                "\n" +
                "确保响应简洁，不超过2000字。\n" +
                "\n" +
                "最后，输出`[FINISH]`。有问题的代码：\n" +
                "```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "需要修改的问题：\n" +
                "1. **移除多余的`@PropertySource`注解**：Spring Boot 自动加载 `src/main/resources` 下的 `application.properties`，无需显式使用 `@PropertySource`，除非有特殊配置文件路径需求。\n" +
                "2. **简化`@EnableAsync`注解**：设置 `proxyTargetClass = true` 会强制使用 CGLIB 代理，这可能增加复杂性。除非业务需要，否则建议移除该参数或设置为 `false`，以保持默认的 JDK 动态代理行为。\n" +
                "3. **改进异常处理**：从 `main` 方法中移除 `throws Exception`，改用 `try-catch` 捕获具体异常（如 `IllegalStateException`），并记录日志或返回错误码，提升启动健壮性。\n" +
                "4. **验证数据源排除的必要性**：排除 `DataSourceAutoConfiguration` 可能导致依赖数据源的组件无法初始化，请确保项目中无数据库依赖，或通过配置文件正确禁用自动配置。\n" +
                "5. **添加日志记录**：建议在 `main` 方法中初始化日志（如使用 SLF4J），以监控启动过程，便于排查问题。\n" +
                "6. **检查依赖冲突**：确保 `@EnableScheduling` 和 `@EnableAsync` 不会与 Spring Boot 的自动配置产生冲突（如重复启用调度器）。\n" +
                "\n" +
                "[FINISH]", stream.getContentBuffer().toString());
        EasyMock.verify(s, c);
    }

    @Test
    public void testAtOnceAndReasonFalse() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(false).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Anthropic_response_atonce.json").openStream(), StandardCharsets.UTF_8);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        stream.atonce(content);
        Assert.assertEquals("有问题的代码：\n" +
                "```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "需要修改的问题：\n" +
                "1. **移除多余的`@PropertySource`注解**：Spring Boot 自动加载 `src/main/resources` 下的 `application.properties`，无需显式使用 `@PropertySource`，除非有特殊配置文件路径需求。\n" +
                "2. **简化`@EnableAsync`注解**：设置 `proxyTargetClass = true` 会强制使用 CGLIB 代理，这可能增加复杂性。除非业务需要，否则建议移除该参数或设置为 `false`，以保持默认的 JDK 动态代理行为。\n" +
                "3. **改进异常处理**：从 `main` 方法中移除 `throws Exception`，改用 `try-catch` 捕获具体异常（如 `IllegalStateException`），并记录日志或返回错误码，提升启动健壮性。\n" +
                "4. **验证数据源排除的必要性**：排除 `DataSourceAutoConfiguration` 可能导致依赖数据源的组件无法初始化，请确保项目中无数据库依赖，或通过配置文件正确禁用自动配置。\n" +
                "5. **添加日志记录**：建议在 `main` 方法中初始化日志（如使用 SLF4J），以监控启动过程，便于排查问题。\n" +
                "6. **检查依赖冲突**：确保 `@EnableScheduling` 和 `@EnableAsync` 不会与 Spring Boot 的自动配置产生冲突（如重复启用调度器）。\n" +
                "\n" +
                "[FINISH]", stream.getContentBuffer().toString());
        EasyMock.verify(s, c);
    }

    @Test
    public void testStream() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(2099), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
            }

            @Override
            public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                return List.of();
            }

            @Override
            public List<TokenData> readAll(Dimension dimension) throws Exception {
                return List.of();
            }

            @Override
            public TokenData read(Dimension dimension, String model) throws Exception {
                return null;
            }

            @Override
            public TokenData read(Dimension dimension) throws Exception {
                return null;
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        File rootDir = new File(ResourceUtils.getURL("classpath:Anthropic_response_stream").getFile());
        File[] dirs = rootDir.listFiles();
        Arrays.sort(dirs, NameFileComparator.NAME_COMPARATOR); // 名字升序排列
        for (File dir : dirs) {
            stream.stream(FileUtils.readFileToString(dir, "UTF-8"));
        }
        Assert.assertEquals("首先，用户要求我作为代码自动化CR助理，检查提交的代码是否有质量问题。指令是：\n" +
                "- 2000字，在结尾处输出`[FINISH]`\n" +
                "- 首先输出有问题的代码，然后输出需要修改的问题，不回答无关内容。\n" +
                "\n" +
                "用户提供了一个Java代码片段：\n" +
                "\n" +
                "```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "我需要检查这个代码是否有质量问题。质量问题可能包括：\n" +
                "- 代码风格\n" +
                "- 最佳实践\n" +
                "- 潜在错误\n" +
                "- 冗余代码\n" +
                "- 安全问题\n" +
                "- 性能问题\n" +
                "- 可维护性问题\n" +
                "\n" +
                "让我分析代码：\n" +
                "- 包声明：`package ai.open.right;` 看起来正常。\n" +
                "- 导入：导入了一些Spring Boot相关的类。看起来正确。\n" +
                "- 注解：\n" +
                "  - `@PropertySource(\"classpath:application.properties\")`：这用于加载属性文件，看起来合理。\n" +
                "  - `@EnableAsync(proxyTargetClass = true)`：启用异步处理，使用CGLIB代理。`proxyTargetClass = true` 是默认值，所以可能冗余，但也可以接受。\n" +
                "  - `@SpringBootApplication(exclude = { DataSourceAutoConfiguration.class })`：排除数据源自动配置，这很常见如果不需要数据库。\n" +
                "- 主类：`MainApplication` 是标准的Spring Boot启动类。\n" +
                "- `main` 方法：调用`SpringApplication.run`，抛出了`Exception`。在main方法中，抛出通用异常是不好的实践，应该捕获或处理特定异常。\n" +
                "\n" +
                "主要问题：\n" +
                "1. **异常处理**：在main方法中，`throws Exception` 是不推荐的。应该捕获异常并适当处理，例如记录日志或显示错误信息。\n" +
                "2. **注解冗余**：`@EnableAsync(proxyTargetClass = true)` 中，`proxyTargetClass = true` 是默认行为，所以可以简化为`@EnableAsync`。\n" +
                "3. **代码格式**：代码格式看起来整洁，但为了完整性，可以检查缩进等。代码中使用了空格缩进，看起来一致。\n" +
                "4. **其他考虑**：没有明显的语法错误或逻辑错误。\n" +
                "\n" +
                "用户要求首先输出有问题的代码，然后输出需要修改的问题。所以，我需要：\n" +
                "- 先复制代码。\n" +
                "- 然后列出问题。\n" +
                "\n" +
                "指令是“不回答无关内容”，所以只输出代码和问题。\n" +
                "\n" +
                "字数要求：2000字。这可能是个误解。指令说“2000字”，但代码和问题可能不多。可能是指整个响应不超过2000字？还是检查代码的内容？我需要确保响应简洁，但达到2000字可能有点多。或许是笔误，意思是“200字”或类似？但用户明确说“2000字”，所以我应该尽量详细，但代码很短。\n" +
                "\n" +
                "重新读指令：“2000字，在结尾处输出`[FINISH]`” – 可能意味着响应应该大约2000字，但代码和问题可能少。代码只有几十行，问题可能几点。为了满足字数，我需要扩展解释，但指令说“不回答无关内容”，所以可能只输出代码和问题，不额外解释。\n" +
                "\n" +
                "用户说：“首先输出有问题的代码，然后输出需要修改的问题” – 所以结构是：\n" +
                "1. 有问题的代码（可能整个代码）\n" +
                "2. 需要修改的问题列表\n" +
                "\n" +
                "然后结尾`[FINISH]`。\n" +
                "\n" +
                "但代码本身可能没有多个问题，我需要列出所有发现的问题。\n" +
                "\n" +
                "或许“输出有问题的代码”意思是指出代码中哪里有问题，而不是复制整个代码。但用户说“输出有问题的代码”，结合上下文，可能是输出整个代码片段。\n" +
                "\n" +
                "为了安全，我输出整个代码，然后列出问题。\n" +
                "\n" +
                "字数：2000字可能是个错误。或许是“200字”或“500字”，但我应该遵循指令。代码和问题可能总共几百字，所以我需要扩展解释每个问题。\n" +
                "\n" +
                "指令说“不回答无关内容”，所以可能只输出代码和问题，不添加额外文本。\n" +
                "\n" +
                "但为了达到2000字，我可能需要详细描述每个问题。但指令说“不需要回答无关内容”，所以可能只输出必要内容。\n" +
                "\n" +
                "再读用户消息：\"首先输出有问题的代码，然后输出需要修改的问题，不回答无关内容\"\n" +
                "\n" +
                "所以，我应该：\n" +
                "- 先输出代码本身（可能整个代码）\n" +
                "- 然后输出问题列表，每个问题描述一下。\n" +
                "\n" +
                "但问题描述可能简短。为了2000字，我需要详细。\n" +
                "\n" +
                "或许“2000字”是针对响应内容，但代码很短，所以可能需要添加解释。但指令说“不回答无关内容”，所以可能只输出代码和问题。\n" +
                "\n" +
                "我认为最好遵守指令：输出代码，然后输出问题。不添加额外文本。\n" +
                "\n" +
                "但字数2000字可能不合理。或许是“200字”的笔误。或者我需要检查代码是否有足够内容。\n" +
                "\n" +
                "另一个想法：用户可能错误地写了“2000字”，但实际期望是标准CR报告。但指令明确。\n" +
                "\n" +
                "我决定：输出整个代码，然后列出发现的问题，每个问题简要描述。为了字数，我可能需要添加一些解释，但严格来说，只输出代码和问题。\n" +
                "\n" +
                "最终输出结构：\n" +
                "- 代码\n" +
                "- 问题列表\n" +
                "- [FINISH]\n" +
                "\n" +
                "问题列表应该清晰。\n" +
                "\n" +
                "列出问题：\n" +
                "1. main方法抛出通用异常。\n" +
                "2. @EnableAsync 注解的 proxyTargetClass 属性冗余。\n" +
                "3. 其他可能问题：如代码没有错误处理，但main方法中抛异常是不好的。\n" +
                "\n" +
                "或许可以添加更多通用建议。\n" +
                "\n" +
                "但为了简洁，只列出现实问题。\n" +
                "\n" +
                "代码中还有其他问题吗？\n" +
                "- 导入：导入了@EnableScheduling，但没有使用。可以删除未使用的导入。\n" +
                "- 属性源：@PropertySource 可能不需要，如果application.properties 是默认的，但这里指定了，可能没问题。\n" +
                "- 排除DataSourceAutoConfiguration 很好，如果不需要数据库。\n" +
                "\n" +
                "所以，主要问题是：\n" +
                "- 异常处理\n" +
                "- 冗余注解\n" +
                "- 未使用导入（@EnableScheduling 导入了但没有使用）\n" +
                "\n" +
                "是的，@EnableScheduling 导入了，但类上没有注解。所以应该删除未使用的导入。\n" +
                "\n" +
                "问题列表：\n" +
                "1. main方法声明 throws Exception，不推荐。\n" +
                "2. @EnableAsync(proxyTargetClass = true) 中 proxyTargetClass = true 是默认，可以省略。\n" +
                "3. 导入了 org.springframework.scheduling.annotation.EnableScheduling 但未使用，应该删除。\n" +
                "4. 可能其他风格问题，但代码看起来整洁。\n" +
                "\n" +
                "现在，输出。\n" +
                "\n" +
                "首先，输出有问题的代码：整个代码。\n" +
                "\n" +
                "然后，输出问题。\n" +
                "\n" +
                "例如：\n" +
                "问题1: main方法抛出通用异常，应该捕获特定异常。\n" +
                "问题2: @EnableAsync 注解可以简化为 @EnableAsync。\n" +
                "问题3: 未使用的导入 EnableScheduling 应该删除。\n" +
                "\n" +
                "为了2000字，我需要详细描述每个问题。但指令说“不回答无关内容”，所以可能只输出简短描述。\n" +
                "\n" +
                "或许“2000字”是错误，我应该忽略字数，专注于内容。\n" +
                "\n" +
                "但用户说“2000字”，所以我需要确保响应长。\n" +
                "\n" +
                "或许代码有问题需要长篇分析，但代码很简单。\n" +
                "\n" +
                "另一个想法：用户可能期望我检查整个代码库，但只提供了一个文件。所以，只分析这个文件。\n" +
                "\n" +
                "我决定输出代码和问题，描述每个问题详细一些，以接近2000字，但可能还是不够。\n" +
                "\n" +
                "代码本身有不到100行。问题描述每个几句话。总共可能500字左右。\n" +
                "\n" +
                "为了达到2000字，我需要添加解释。\n" +
                "\n" +
                "但指令说“不回答无关内容”，所以可能只输出代码和问题。\n" +
                "\n" +
                "或许“输出有问题的代码”意思是指出代码行有问题的部分，而不是整个代码。\n" +
                "\n" +
                "但用户说“输出有问题的代码”，可能整个代码。\n" +
                "\n" +
                "为了安全，我输出整个代码。\n" +
                "\n" +
                "最终计划：\n" +
                "1. 复制代码。\n" +
                "2. 列出问题，每个问题用几句话描述。\n" +
                "3. [FINISH]\n" +
                "\n" +
                "字数可能不足，但尽力而为。\n" +
                "\n" +
                "开始写响应。```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "问题描述：\n" +
                "1. **异常处理不当**：main 方法声明了 `throws Exception`，这会抛出通用异常，不利于错误定位和维护。应该捕获具体异常并处理，例如使用 try-catch 块记录日志或显示错误信息。\n" +
                "2. **冗余注解配置**：`@EnableAsync(proxyTargetClass = true)` 中的 `proxyTargetClass = true` 是默认值，可以简化为 `@EnableAsync`，以减少冗余代码。\n" +
                "3. **未使用的导入**：导入了 `org.springframework.scheduling.annotation.EnableScheduling`，但未在类上使用该注解。建议删除未使用的导入，以保持代码整洁。\n" +
                "4. **代码风格可优化**：尽管代码格式基本正确，但为了提高可读性，可以考虑在注解和括号周围添加适当的空格，例如 `@EnableAsync` 和 `@SpringBootApplication` 之间保持一致缩进。\n" +
                "[FINISH]", stream.getContentBuffer().toString());
    }

    @Test
    public void testStreamWithOutReason() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildNotifierManagerWithimplement();
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getPrintReason()).andReturn(false).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {
                Assert.assertEquals(Integer.valueOf(2099), tokenData.getTotal());
                Assert.assertEquals(Integer.valueOf(0), tokenData.getCache());
            }

            @Override
            public List<TokenData> readAll(Dimension dimension, List<String> model) throws Exception {
                return List.of();
            }

            @Override
            public List<TokenData> readAll(Dimension dimension) throws Exception {
                return List.of();
            }

            @Override
            public TokenData read(Dimension dimension, String model) throws Exception {
                return null;
            }

            @Override
            public TokenData read(Dimension dimension) throws Exception {
                return null;
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        })
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }
        };
        File rootDir = new File(ResourceUtils.getURL("classpath:Anthropic_response_stream").getFile());
        File[] dirs = rootDir.listFiles();
        Arrays.sort(dirs, NameFileComparator.NAME_COMPARATOR); // 名字升序排列
        for (File dir : dirs) {
            stream.stream(FileUtils.readFileToString(dir, "UTF-8"));
        }
        Assert.assertEquals("```java\n" +
                "package ai.open.right;\n" +
                "\n" +
                "import org.springframework.boot.SpringApplication;\n" +
                "import org.springframework.boot.autoconfigure.SpringBootApplication;\n" +
                "import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;\n" +
                "import org.springframework.context.annotation.PropertySource;\n" +
                "import org.springframework.scheduling.annotation.EnableAsync;\n" +
                "import org.springframework.scheduling.annotation.EnableScheduling;\n" +
                "\n" +
                "@PropertySource(\"classpath:application.properties\")\n" +
                "@EnableAsync(proxyTargetClass = true)\n" +
                "@SpringBootApplication(exclude = {\n" +
                "        DataSourceAutoConfiguration.class\n" +
                "})\n" +
                "public class MainApplication {\n" +
                "\n" +
                "    public static void main(String[] args) throws Exception {\n" +
                "        SpringApplication.run(MainApplication.class, args);\n" +
                "    }\n" +
                "}\n" +
                "```\n" +
                "\n" +
                "问题描述：\n" +
                "1. **异常处理不当**：main 方法声明了 `throws Exception`，这会抛出通用异常，不利于错误定位和维护。应该捕获具体异常并处理，例如使用 try-catch 块记录日志或显示错误信息。\n" +
                "2. **冗余注解配置**：`@EnableAsync(proxyTargetClass = true)` 中的 `proxyTargetClass = true` 是默认值，可以简化为 `@EnableAsync`，以减少冗余代码。\n" +
                "3. **未使用的导入**：导入了 `org.springframework.scheduling.annotation.EnableScheduling`，但未在类上使用该注解。建议删除未使用的导入，以保持代码整洁。\n" +
                "4. **代码风格可优化**：尽管代码格式基本正确，但为了提高可读性，可以考虑在注解和括号周围添加适当的空格，例如 `@EnableAsync` 和 `@SpringBootApplication` 之间保持一致缩进。\n" +
                "[FINISH]", stream.getContentBuffer().toString());
    }

    @Test
    public void testFunCallAtOnce() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getFunCallHeritage()).andReturn(false).anyTimes();
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(5000).anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(null).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(c.isTakeover("Workflow_funcall_workflow__materials")).andReturn(false).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        String content = IOUtils.toString(ResourceUtils.getURL("classpath:Anthropic_response_funcall_atonce.json").openStream(), StandardCharsets.UTF_8);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected void notifySegment() throws Exception {

            }
        };
        stream.atonce(content);
        Assert.assertEquals("用户想了解第10周可以烹饪哪些食材，以及如何烹饪这些食材。我需要先调用第一个工具来获取第10周的食材，然后再调用第二个工具来获取这些食材的烹饪方法。\n" +
                "\n" +
                "我应该先调用 Workflow_funcall_workflow__materials 来获取第10周的食材。我来帮您查询第10周可以烹饪的食材和烹饪方法。\n" +
                "OK", stream.getContentBuffer().toString());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(stream.getProviderFunRequests().size()));
        Assert.assertEquals("call_function_ifybzsyc7ss5_1", stream.getProviderFunRequests().getFirst().getId());
        Assert.assertEquals("Workflow_funcall_workflow__materials", stream.getProviderFunRequests().getFirst().getName());
        Assert.assertEquals("{\"week\":10}", JsonUtils.write(stream.getProviderFunRequests().getFirst().getArgs()));
        Assert.assertEquals("{\"type\":\"tool_use\",\"id\":\"call_function_ifybzsyc7ss5_1\",\"name\":\"Workflow_funcall_workflow__materials\",\"input\":{\"week\":10}}", JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()));
    }

    @Test
    public void testFunCallStream() throws Exception {
        NotifierServiceImpl r = ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("OK");
        SignalStream s = EasyMock.createMock(SignalStream.class);
        HistoryStore h = ObjectBuilder.buildMockHistoryWithStore();
        AnthropicRequest c = EasyMock.createMock(AnthropicRequest.class);
        EasyMock.expect(c.getFunCallHeritage()).andReturn(false).anyTimes();
        EasyMock.expect(c.getModel()).andReturn("HELLO").anyTimes();
        EasyMock.expect(c.getApi()).andReturn("WORLD").anyTimes();
        EasyMock.expect(c.getFunCallTimeout()).andReturn(5000).anyTimes();
        EasyMock.expect(c.getMetadata()).andReturn(null).anyTimes();
        EasyMock.expect(c.getStoreFunCall()).andReturn(false).anyTimes();
        EasyMock.expect(c.isTakeover("Workflow_funcall_workflow__materials")).andReturn(false).anyTimes();
        EasyMock.expect(c.getPrintReason()).andReturn(true).anyTimes();
        EasyMock.expect(c.getScene()).andReturn("WORKFLOW").anyTimes();
        s.signal(EasyMock.anyObject(SignalExecutor.class), EasyMock.anyObject(Message.class));
        EasyMock.expectLastCall().anyTimes();
        EasyMock.expect(c.getQuery4History()).andReturn("UNKNOWN").anyTimes();
        EasyMock.expect(c.getRepositories()).andReturn(Arrays.asList("UNKNOWN")).anyTimes();
        EasyMock.expect(c.getExpired()).andReturn(1000).anyTimes();
        EasyMock.expect(c.isWriteable()).andReturn(true).anyTimes();
        EasyMock.expect(c.hasChain()).andReturn(true).anyTimes();
        EasyMock.expect(c.getChain()).andReturn("NEXT_WORKFLOW").anyTimes();
        EasyMock.expect(c.getTokenFirst()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getTokenBuffer()).andReturn(1024).anyTimes();
        EasyMock.expect(c.getStream()).andReturn(false).anyTimes();
        EasyMock.expect(c.getMessage()).andReturn(Message.build(ObjectBuilder.buildLLMQuery())).anyTimes();
        EasyMock.expect(c.getContainHistories()).andReturn(true).anyTimes();
        EasyMock.expect(c.getHistories()).andReturn(null).anyTimes();
        EasyMock.expect(c.getPrefix()).andReturn("").anyTimes();
        EasyMock.expect(c.getSuffix()).andReturn("").anyTimes();
        EasyMock.expect(c.getNotifier(Notifier.LOCALHOST)).andReturn(Notifier.LOCALHOST).anyTimes();
        EasyMock.expect(c.hasNotifier()).andReturn(false).anyTimes();
        EasyMock.replay(s, c);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(r)
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(s)
                .historyStore(h)
                .namesService(ObjectBuilder.buildNamesService())
                .request(c)
                .build()) {
            @Override
            protected void storeConversation(String content) throws Exception {
            }

            @Override
            protected void notifySegment() throws Exception {

            }
        };
        File rootDir = new File(ResourceUtils.getURL("classpath:Anthropic_response_funcall_stream").getFile());
        File[] dirs = rootDir.listFiles();
        Arrays.sort(dirs, NameFileComparator.NAME_COMPARATOR); // 名字升序排列
        for (File dir : dirs) {
            stream.stream(FileUtils.readFileToString(dir, "UTF-8"));
        }
        Assert.assertEquals("用户说今天是第10周，想知道可以烹饪哪些食材以及如何烹饪。\n" +
                "\n" +
                "我需要：\n" +
                "1. 首先调用 `Workflow_funcall_workflow__materials` 来获取第10周的食材\n" +
                "2. 然后根据返回的食材，再调用 `Workflow_funcall_workflow__cooking` 来获取烹饪方法\n" +
                "3. 最后将结果整理后直接告诉用户，不要再问用户问题\n" +
                "\n" +
                "我应该连续调用两个工具。我来帮您查询第10周的食材和烹饪方法。\n" +
                "OK", stream.getContentBuffer().toString());
        Assert.assertEquals(Integer.valueOf(1), Integer.valueOf(stream.getProviderFunRequests().size()));
        Assert.assertEquals("call_function_o218h2t0pp0r_1", stream.getProviderFunRequests().getFirst().getId());
        Assert.assertEquals("Workflow_funcall_workflow__materials", stream.getProviderFunRequests().getFirst().getName());
        Assert.assertEquals("{\"week\":10}", JsonUtils.write(stream.getProviderFunRequests().getFirst().getArgs()));
        Assert.assertEquals("{\"type\":\"tool_use\",\"id\":\"call_function_o218h2t0pp0r_1\",\"name\":\"Workflow_funcall_workflow__materials\",\"input\":{\"week\":10}}", JsonUtils.write(stream.getProviderFunRequests().getFirst().getRefer()));
    }

    /**
     * 覆盖 {@link AnthropicStream#cacheStatistic(Map)}：流式 message_start 的 usage 位于 message.usage，
     * 应能正确提取 cache 和 total。
     */
    @Test
    public void testCacheStatistic_readsMessageStartUsage() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("input_tokens", 60);
        usage.put("output_tokens", 0);
        usage.put("cache_creation_input_tokens", 10);
        usage.put("cache_read_input_tokens", 25);
        Map<String, Object> message = new HashMap<>();
        message.put("usage", usage);
        Map<String, Object> body = new HashMap<>();
        body.put("type", "message_start");
        body.put("message", message);
        stream.cacheStatistic(body);
        Assert.assertNotNull(stream.tokenData);
        Assert.assertEquals(Integer.valueOf(95), stream.tokenData.getInput());
        Assert.assertEquals(Integer.valueOf(35), stream.tokenData.getCache());
        Assert.assertEquals(Integer.valueOf(95), stream.tokenData.getTotal());
    }

    /**
     * 覆盖 {@link AnthropicStream#tokenStatistic(Map)}：先经 cacheStatistic 生成 tokenData，
     * 再写入 segment.usage（cache、total）。
     */
    @Test
    public void testTokenStatistic_withUsage_callsStatAndSetsSegmentUsage() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("input_tokens", 60);
        usage.put("output_tokens", 0);
        usage.put("cache_read_input_tokens", 25);
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.cacheStatistic(body);
        stream.tokenStatistic(body);
        Assert.assertNotNull(stream.getSegment().getUsage());
        Assert.assertEquals(Integer.valueOf(25), stream.getSegment().getUsage().getCache());
        Assert.assertEquals(Integer.valueOf(85), stream.getSegment().getUsage().getTotal());
    }

    /**
     * 覆盖 {@link AnthropicStream#tokenStatistic(Map)}：cache==0 且 total==0 时不设置 segment.usage。
     */
    @Test
    public void testTokenStatistic_zeroCacheAndTotal_doesNotStat() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(null)
                .providerReason(null)
                .signalStream(null)
                .historyStore(null)
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());
        Map<String, Object> usage = new HashMap<>();
        usage.put("input_tokens", 0);
        usage.put("output_tokens", 0);
        usage.put("cache_read_input_tokens", 0);
        Map<String, Object> body = new HashMap<>();
        body.put("usage", usage);
        stream.cacheStatistic(body);
        stream.tokenStatistic(body);
        Assert.assertNull(stream.getSegment().getUsage());
    }

    /**
     * 覆盖 {@link AnthropicStream#stream(String)}：去掉空行后若 source 为空则早退返回 true。
     */
    @Test
    public void testStream_emptyAfterBlankLineFilter_returnsTrue() throws Exception {
        AnthropicRequest request = new AnthropicRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        AnthropicStream stream = new AnthropicStream(ProviderStreamConfig.<AnthropicRequest>builder()
                .trackFunCallService(null)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildNotifierManagerWithimplement())
                .providerReason(null)
                .signalStream(null)
                .historyStore(ObjectBuilder.buildMockHistoryWithStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());

        Assert.assertTrue("empty string", stream.stream(""));
        Assert.assertTrue("only spaces", stream.stream("   "));
        Assert.assertTrue("only newlines", stream.stream("\n\n"));
        Assert.assertTrue("blank lines only", stream.stream("  \n  \n  "));
    }
}
