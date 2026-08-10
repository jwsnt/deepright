package ai.open.right.integration;

import ai.open.right.ObjectBuilder;
import ai.open.right.context.UserContext;
import ai.open.right.protocol.Protocol;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

class RightTaskGetSetTest {

    private RightConfig rightConfig;
    private RightTask rightTask;

    @BeforeEach
    void setUp() {
        rightConfig = RightConfig.builder().build();
        rightConfig.setQuery("test query");
        rightConfig.setConversation("conv-123");
        rightConfig.setChat("chat-456");
        rightConfig.setProtocol(Protocol.CHAT);
        rightConfig.setWorkflow("workflow-abc");
        rightConfig.setTrace("trace-" + UUID.randomUUID());
        rightConfig.setBiz("biz-test");
        rightConfig.setUserContext(new UserContext());
        rightConfig.setMetadata(new HashMap<>(Map.of("key1", "value1")));
        rightConfig.setHistories(List.of(new History()));
        rightConfig.setMediaContext(List.of(new MediaContext()));

        rightTask = new RightTask(rightConfig, ObjectBuilder.buildNotifyWriteBack());
    }

    @Test
    void testInitialization() {
        assertNotNull(rightTask.getCreated());
        assertEquals("test query", rightTask.getQuery());
        assertEquals("conv-123", rightTask.getConversation());
        assertEquals("chat-456", rightTask.getChat());
        assertEquals(Protocol.CHAT, rightTask.getProtocol());
        assertEquals("workflow-abc", rightTask.getWorkflow());
        assertEquals("biz-test", rightTask.getBiz());
        assertNotNull(rightTask.getUserContext());
        assertEquals(null, rightTask.getDeepness());
    }

    @Test
    void testSetUserContext() {
        UserContext newUserContext = new UserContext();
        rightTask.setUserContext(newUserContext);
        assertSame(newUserContext, rightTask.getUserContext());
    }

    @Test
    void testSetHistories() {
        List<History> histories = List.of(new History());
        rightTask.addHistories(histories);
        assertNotSame(histories, rightTask.getHistories());
    }

    @Test
    void testSetWorkflow() {
        String newWorkflow = "new-workflow";
        rightTask.setWorkflow(newWorkflow);
        assertEquals(newWorkflow, rightTask.getWorkflow());
    }

    @Test
    void testSetNotifier() {
        String newNotifier = "new-notifier";
        rightTask.setNotifier(newNotifier);
        assertEquals(newNotifier, rightTask.getNotifier());
    }

    @Test
    void testSetDeepness() {
        rightTask.setDeepness(2);
        assertEquals(2, rightTask.getDeepness());
        assertEquals(2, rightConfig.getDeepness());
        rightTask.setDeepness(5);
        assertEquals(5, rightTask.getDeepness());
    }

    @Test
    void testSetProtocol() {
        String newProtocol = Protocol.CHAT;
        rightTask.setProtocol(newProtocol);
        assertEquals(newProtocol, rightTask.getProtocol());
    }

    @Test
    void testSetUpstream() {
        String newUpstream = "new-upstream";
        rightTask.setUpstream(newUpstream);
        assertEquals(newUpstream, rightTask.getUpstream());
    }

    @Test
    void testSetQuery() {
        String newQuery = "new query";
        rightTask.setQuery(newQuery);
        assertEquals(newQuery, rightTask.getQuery());
    }

    @Test
    void testSetBiz() {
        String newBiz = "new-biz";
        rightTask.setBiz(newBiz);
        assertEquals(newBiz, rightTask.getBiz());
    }

    @Test
    void testSetTrace() {
        String newTrace = "new-trace";
        rightTask.setTrace(newTrace);
        assertEquals(newTrace, rightTask.getTrace());
    }

    @Test
    void testSetProviderAndToken() {
        rightTask.setProviderAndToken("provider-x", "token-y");

        assertEquals("provider-x", rightTask.getMetadata().get(ProviderRequestService.KEY_PROVIDER));
        assertEquals(
                "token-y",
                rightTask.getMetadata().get(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN)
        );
    }

    @Test
    void testFunCallTrack() {
        String trackId = UUID.randomUUID().toString();
        rightTask.beginFunCallTrack(trackId);
        assertEquals(trackId, rightTask.getFunCallTrack());
        assertTrue(rightTask.containFunCallTrack());
        rightTask.closeFunCallTrack();
        assertNull(rightTask.getFunCallTrack());
        assertFalse(rightTask.containFunCallTrack());
    }

    @Test
    void testChatTrack() {
        rightTask.beginChatTrack();
        assertTrue(rightTask.containChatTrack());
    }

    @Test
    void testInit() {
        RightConfig emptyConfig = RightConfig.builder().build();
        RightTask emptyTask = new RightTask(emptyConfig, ObjectBuilder.buildNotifyWriteBack());
        emptyTask.init();
        assertNotNull(emptyTask.getConversation());
        assertNotNull(emptyTask.getProtocol());
        assertNotNull(emptyTask.getChat());
        assertEquals(Protocol.CHAT, emptyTask.getProtocol());
    }
}
