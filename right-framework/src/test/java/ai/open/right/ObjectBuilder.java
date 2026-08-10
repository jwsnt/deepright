package ai.open.right;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.listener.Event;
import ai.open.right.listener.impl.EventListenerServiceImpl;
import ai.open.right.netty.NettyCaptor;
import ai.open.right.netty.chat.NettySegment;
import ai.open.right.netty.chat.distribute.NettyRequest;
import ai.open.right.netty.chat.server.http.NettyErrorUsage;
import ai.open.right.netty.mcp.server.NettyMcpRequest;
import ai.open.right.protocol.Protocol;
import ai.open.right.release.ResourceReleaser;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.resouce.ResourceService;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.config.impl.NamesServiceImpl;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.config.McpConfig;
import ai.open.right.workflow.flow.file.DefStore;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.llm.*;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.llm.store.history.HistoryPair;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.flow.media.MediaInlineData;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.server.McpRequest;
import ai.open.right.workflow.notify.NothingWriteBack;
import ai.open.right.workflow.notify.Notifier;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.impl.NotifierServiceImpl;
import com.google.common.collect.ImmutableMap;
import io.netty.channel.ChannelHandlerContext;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.springframework.util.ResourceUtils;

import java.io.File;
import java.net.URL;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;

public class ObjectBuilder {

    public static DefStore buildDefStore() throws Exception {
        DefStore defStore = new DefStore();
        defStore.setFileStore(ImmutableMap.of(SysStore.NAME, ObjectBuilder.buildFileStore()));
        return defStore;
    }

    public static FileStore buildFileStore() throws Exception {
        SysStore fileStore = new SysStore() {

            @Override
            public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
                String resource = super.store(bytes, suffix, workTask);
                new File(resource).delete();
                return resource;
            }
        };
        fileStore.setOversize(10485760);
        fileStore.setDeleteOnExit(true);
        fileStore.setPath(".");
        fileStore.init();
        return fileStore;
    }

    public static ResourceReleaser buidResourceConfig() {
        return new ResourceReleaser() {
            @Override
            public String getRoot() throws Exception {
                return new File(".").getAbsolutePath();
            }
        };
    }


    public static ResourceService buildResourceService(Class clazz) {
        return new ResourceService() {

            @Override
            public URL url(String location) throws Exception {
                return ResourceUtils.getURL(location);
            }

            @Override
            public Class<?> root() throws Exception {
                return clazz;
            }
        };
    }

    public static ResourceService buildResourceService() {
        return new ResourceService() {

            @Override
            public URL url(String location) throws Exception {
                return ResourceUtils.getURL(location);
            }

            @Override
            public Class<?> root() throws Exception {
                return ObjectBuilder.class;
            }
        };
    }

    public static MediaInlineService buildMediaInlineService(String path) {
        return new MediaInlineService() {

            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                return path;
            }
        };
    }

    public static MediaInlineService buildMediaInlineService() {
        return new MediaInlineService() {

            @Override
            public String write(MediaInlineData mediaInlineData, WorkflowTask workTask) throws Exception {
                return "";
            }
        };
    }

    public static TokenStatistic buildTokenStatistic() {
        return new TokenStatistic() {

            @Override
            public void stat(ProviderRequest providerRequest, TokenData tokenData) throws Exception {

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
                return TokenData.builder().build();
            }

            @Override
            public Set<String> models() throws Exception {
                return Set.of();
            }
        };
    }

    public static WorkflowTask buildWorkflowTask() {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        nettyRequest.setMetadata(new HashMap<String, Object>());
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(RedirectContext.DEEPNESS);
        return nettyRequest;
    }

    public static WorkflowTask buildWorkflowTaskWithOutWrite() {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void writeSource(Segment segment) throws Exception {

            }

            @Override
            public void writeBack(Segment segment) throws Exception {
            }

            @Override
            public void close() throws Exception {
            }
        };
        nettyRequest.setMetadata(new HashMap<String, Object>());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(RedirectContext.DEEPNESS);
        return nettyRequest;
    }

    public static WorkflowTask buildWorkflowTaskWithTimestamp(Long timestamp) {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        nettyRequest.setMetadata(new HashMap<String, Object>());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setCreated(timestamp);
        nettyRequest.setDeepness(0);
        return nettyRequest;
    }

    public static LLMQuery buildLLMQueryWithBiz(String biz) {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        Map<String, Object> meta = new HashMap<String, Object>();
        meta.put("KEY1", "VAL1");
        nettyRequest.setMetadata(meta);
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(0);
        nettyRequest.setBiz(biz);
        return new LLMQueryDelegate(nettyRequest, nettyRequest.getWorkflow(), nettyRequest.getNotifier());
    }

    public static LLMQuery buildLLMQueryWithEmptyMetadata() {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        Map<String, Object> meta = new HashMap<String, Object>();
        nettyRequest.setMetadata(meta);
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(0);
        return new LLMQueryDelegate(nettyRequest, nettyRequest.getWorkflow(), nettyRequest.getNotifier());
    }

    public static LLMQuery buildLLMQuery(Map<String, Object> meta) {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        nettyRequest.setMetadata(meta);
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(0);
        return new LLMQueryDelegate(nettyRequest, nettyRequest.getWorkflow(), nettyRequest.getNotifier());
    }

    public static LLMQuery buildLLMQuery() {
        NettyRequest nettyRequest = new NettyRequest() {
            @Override
            public void checkClosed() throws Exception {
            }

            @Override
            public Boolean isClosed() throws Exception {
                return false;
            }

            @Override
            public void close() throws Exception {
            }
        };
        Map<String, Object> meta = new HashMap<String, Object>();
        meta.put("KEY1", "VAL1");
        nettyRequest.setMetadata(meta);
        nettyRequest.setCreated(System.currentTimeMillis());
        nettyRequest.setUserContext(ObjectBuilder.buildEmpty());
        nettyRequest.setConversation(NettyRequest.UNKNOWN);
        nettyRequest.setUpstream(NettyRequest.UNKNOWN);
        nettyRequest.setWorkflow(NettyRequest.UNKNOWN);
        nettyRequest.setNotifier(Notifier.LOCALHOST);
        nettyRequest.setTrace(NettyRequest.UNKNOWN);
        nettyRequest.setQuery(NettyRequest.UNKNOWN);
        nettyRequest.setChat(NettyRequest.UNKNOWN);
        nettyRequest.setBiz(NettyRequest.UNKNOWN);
        nettyRequest.setProtocol(Protocol.CHAT);
        nettyRequest.setOriginal("ORIGINAL");
        nettyRequest.setPrevious("PREVIOUS");
        nettyRequest.setInitial("INITIAL");
        nettyRequest.setDeepness(0);
        return new LLMQueryDelegate(nettyRequest, nettyRequest.getWorkflow(), nettyRequest.getNotifier());
    }

    public static Segment buildSegment(Segment.SegmentConfig config) {
        return Segment.build(ObjectBuilder.buildWorkflowTask(), config);
    }

    public static Segment buildSegmentWithOutFinish() {
        return Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .finished(false)
                .build());
    }

    public static Segment buildSegment(Integer code) {
        return Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder()
                .code(code)
                .build());
    }

    public static Segment buildSegment() {
        return Segment.build(ObjectBuilder.buildWorkflowTask(), Segment.SegmentConfig.builder().build());
    }

    public static NotifierServiceImpl buildNotifierManagerWithimplement() throws Exception {
        NotifierServiceImpl notifierManager = new NotifierServiceImpl() {
            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {

            }
        };
        return notifierManager;
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithWriteBackContent(String content) {
        return new NotifierServiceImpl() {

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setContent(content);
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setContent(content);
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setContent(content);
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setContent(content);
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }
        };
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithMediaContext() {
        return new NotifierServiceImpl() {

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(Notifier.LOCALHOST, segment.getNotifier());
            }
        };
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithWriteBackContent(Object... content) {
        return new NotifierServiceImpl() {

            private AtomicInteger i = new AtomicInteger(0);

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Object each = content[i.getAndIncrement()];
                if (Exception.class.isAssignableFrom(each.getClass())) {
                    throw Exception.class.cast(each);
                } else {
                    segment.setContent(each.toString());
                    segment.setUsage(new SegmentUsage());
                    notifierWriteBack.writeBack(segment);
                }
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Object each = content[i.getAndIncrement()];
                if (Exception.class.isAssignableFrom(each.getClass())) {
                    throw Exception.class.cast(each);
                } else {
                    segment.setContent(each.toString());
                    segment.setUsage(new SegmentUsage());
                    notifierWriteBack.writeBack(segment);
                }
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Object each = content[i.getAndIncrement()];
                if (Exception.class.isAssignableFrom(each.getClass())) {
                    throw Exception.class.cast(each);
                } else {
                    segment.setContent(each.toString());
                    segment.setUsage(new SegmentUsage());
                    notifierWriteBack.writeBack(segment);
                }
            }
        };
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithWriteBackException() {
        return new NotifierServiceImpl() {

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                throw new RuntimeException();
            }

            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                throw new RuntimeException();
            }

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                throw new RuntimeException();
            }
        };
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithWriteBackDirect() {
        return new NotifierServiceImpl() {

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }
        };
    }

    public static NotifierServiceImpl buildAssertNotifierManagerWithOnlyAssert(String content) {
        return new NotifierServiceImpl() {
            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
                segment.setUsage(new SegmentUsage());
                Assert.assertEquals(content, segment.getContent());
            }
        };
    }

    public static NotifierServiceImpl buildAssertNotifierManagerWithWriteBackDirect(String content) {
        return new NotifierServiceImpl() {
            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {
                Assert.assertEquals(content, segment.getContent());
                segment.setUsage(new SegmentUsage());
                notifierWriteBack.writeBack(segment);
            }
        };
    }

    public static NotifierServiceImpl buildActualNotifierManagerWithNothing() {
        return new NotifierServiceImpl() {

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            public void notify(String notifier, Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }

            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {
            }

            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {
            }
        };
    }


    public static HistoryStore buildMockHistoryWithNothing() {
        return EasyMock.createMock(HistoryStore.class);
    }

    public static HistoryStore buildMockHistoryWithStore() throws Exception {
        HistoryStore h = EasyMock.createMock(HistoryStore.class);
        h.store(EasyMock.anyObject(), EasyMock.anyObject(List.class), EasyMock.anyString(), EasyMock.anyObject(), EasyMock.anyInt(), EasyMock.anyInt(), EasyMock.anyLong());
        EasyMock.expectLastCall();
        return h;
    }

    public static Dimension buildDimension() {
        return new Dimension() {
            @Override
            public String getBiz() {
                return "BIZ";
            }

            @Override
            public String getChat() {
                return "CHAT";
            }

            @Override
            public String getDevice() {
                return "DEVICE";
            }

            @Override
            public String getWorkflow() {
                return "WORKFLOW";
            }

            @Override
            public String getDimension() {
                return this.getBiz() + this.getChat() + this.getDevice();
            }
        };
    }

    public static Event buildEvent() {

        return new Event() {

            @Override
            public String getDevice() {
                return "DEVICE";
            }

            @Override
            public String getWorkflow() {
                return "WORKFLOW";
            }

            @Override
            public String getDimension() {
                return this.getBiz() + this.getChat() + this.getDevice();
            }

            @Override
            public String getChat() {
                return "CHAT";
            }

            @Override
            public String getType() {
                return "TYPE";
            }

            @Override
            public Object getData() {
                return "DATA";
            }

            @Override
            public String getBiz() {
                return "BIZ";
            }

            @Override
            public Long getNow() {
                return 10086L;
            }

            @Override
            public Event init() {
                return this;
            }
        };
    }

    public static EventListenerServiceImpl buildEventListenerService() {
        return new EventListenerServiceImpl();
    }

    public static NettySegment buildEmptyNettySegment() {

        return new NettySegment() {

            @Override
            public Map<String, Object> getMetadata() {
                return null;
            }

            @Override
            public LLMUsage getUsage() {
                return new NettyErrorUsage();
            }

            @Override
            public String getTrace() {
                return null;
            }

            @Override
            public Integer getCode() {
                return 200;
            }

            @Override
            public String getBiz() {
                return null;
            }

            @Override
            public String getId() {
                return "";
            }

            @Override
            public Integer getIndex() {
                return null;
            }

            @Override
            public Long getTimestamp() {
                return null;
            }

            @Override
            public String getContent() {
                return null;
            }

            @Override
            public String getWorkflow() {
                return null;
            }

            @Override
            public Boolean getStream() {
                return true;
            }

            @Override
            public Boolean isFinished() {
                return true;
            }

            @Override
            public void mark() {

            }
        };
    }

    public static PlaceholderResolver buildEmptyPlaceholderResolver() {
        return new PlaceholderResolver() {

            @Override
            public String replace(String input) {
                return input;
            }
        };
    }

    public static NotifierWriteBack buildNotifyWriteBack() {
        return new NothingWriteBack();
    }

    public static NotifierWriteBack buildNotifyWriteBackWithWriteException() {
        return new NothingWriteBack() {
            @Override
            public void writeSource(Segment segment) throws Exception {
                throw new RuntimeException();
            }

            @Override
            public void writeBack(Segment segment) throws Exception {
                throw new RuntimeException();
            }
        };
    }


    public static Notifier buildActualNotifierWithNothing() {
        return new Notifier() {

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, RedirectContext redirectContext, NotifierWriteBack notifierWriteBack) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack, List<MediaContext> mediaContext) throws Exception {

            }

            @Override
            public void notify(Segment segment, NotifierWriteBack notifierWriteBack) throws Exception {

            }
        };
    }

    public static NettyCaptor buildNettyExpCaptor() {
        return new NettyCaptor() {

            @Override
            public void exceptionCaught(ChannelHandlerContext ctx, Throwable e) throws Exception {

            }
        };
    }

    public static McpDimension buildMcpDimensionWithMcpConfig() {
        return McpDimension.builder()
                .chat("Chat")
                .biz("Biz")
                .workflow("Workflow")
                .device("Device")
                .mcpConfig(new McpConfig())
                .build();
    }

    public static NamesService buildNamesService() throws Exception {
        NamesServiceImpl namesService = new NamesServiceImpl();
        namesService.init();
        return namesService;
    }

    public static McpRequest buildEmptyMcpRequest() {
        Map<String, Object> content = new HashMap<>();
        content.put("params", ImmutableMap.of("text", "HELLO WORLD", "name", "NAME"));
        Map<String, String> header = new HashMap<>();
        return NettyMcpRequest.builder()
                .content(content)
                .headers(header)
                .trace("TRACE")
                .build();
    }

    public static UserContext buildEmpty() {
        return UserContext.builder()
                .language(UserContext.UNKNOWN)
                .device(UserContext.UNKNOWN)
                .region(UserContext.UNKNOWN)
                .system(UserContext.UNKNOWN)
                .model(UserContext.UNKNOWN)
                .token(UserContext.UNKNOWN)
                .brand(UserContext.UNKNOWN)
                .build();
    }

    public static HistoryStore buildHistoryStore() {
        return new HistoryStore() {

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, String reasoning, Integer expire, Integer nums, Long now) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, String query, String answer, Integer expire, Integer nums, Long now) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, List<HistoryPair> pairs, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public void store(Dimension dimension, List<String> repositories, HistoryPair pair, Integer expire, Integer nums) throws Exception {

            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now, Long offset) throws Exception {
                return List.of();
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc, Long now) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Boolean desc) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums, Long now) throws Exception {
                return null;
            }

            @Override
            public List<History> restore(Dimension dimension, String scene, Integer nums) throws Exception {
                return null;
            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Boolean desc, Long now) throws Exception {

            }

            @Override
            public void clear(Dimension dimension, List<String> repositories, Long now) throws Exception {

            }
        };
    }

    public static ProviderReason getProviderReason() {
        return new ProviderReason() {

            @Override
            public String reason(ProviderRequest request, String message, Boolean finished, Integer index) throws Exception {
                return message;
            }
        };
    }
}
