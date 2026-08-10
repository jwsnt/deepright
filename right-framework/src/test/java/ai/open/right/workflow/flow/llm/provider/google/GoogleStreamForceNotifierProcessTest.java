package ai.open.right.workflow.flow.llm.provider.google;

import ai.open.right.ObjectBuilder;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import com.google.common.collect.ImmutableMap;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Field;
import java.util.*;

import ai.open.right.workflow.flow.llm.provider.ProviderStreamConfig;
/**
 * 单测：GoogleStream#forceNotifierProcess 置位、复位及复位后走 notifySegment。
 * 独立成类避免依赖 GoogleStreamTest 的 setUp（EasyMock 在部分环境对 GoogleRequest 的 mock 会失败）。
 */
public class GoogleStreamForceNotifierProcessTest {

    /**
     * stream() 遇到 functionCall 时置位 forceNotifierProcess 并转交 atonce；验证置位为 true。
     */
    @Test
    public void testForceNotifierProcessSetWhenFunctionCallInStream() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithNothing())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build());

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        Map<String, Object> candidate = new HashMap<>();
        Map<String, Object> content = new HashMap<>();
        content.put("role", "model");
        List<Map<String, Object>> parts = new ArrayList<>();
        Map<String, Object> part = new HashMap<>();
        part.put("functionCall", ImmutableMap.of("name", "get_weather", "args", new HashMap<String, Object>()));
        parts.add(part);
        content.put("parts", parts);
        candidate.put("content", content);
        candidates.add(candidate);
        body.put("candidates", candidates);

        stream.stream(JsonUtils.write(body));

        Field f = GoogleStream.class.getDeclaredField("forceNotifierProcess");
        f.setAccessible(true);
        Assertions.assertTrue(Boolean.TRUE.equals(f.get(stream)), "forceNotifierProcess should be true after stream() with functionCall");
    }

    /**
     * stream() 在 totalFinish 且 forceNotifierProcess 为 true 时调用 notifyProcess 并复位标志；验证复位。
     */
    @Test
    public void testForceNotifierProcessResetWhenTotalFinishInStream() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setTokenFirst(1024);
        request.setTokenBuffer(1024);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);

        final boolean[] notifyProcessCalled = {false};
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithNothing())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected void notifyProcess() throws Exception {
                notifyProcessCalled[0] = true;
            }

            @Override
            protected void notifySegment() throws Exception {
            }
        };

        Field f = GoogleStream.class.getDeclaredField("forceNotifierProcess");
        f.setAccessible(true);
        f.set(stream, true);

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        Map<String, Object> candidate = new HashMap<>();
        candidate.put("finishReason", "STOP");
        Map<String, Object> content = new HashMap<>();
        content.put("role", "model");
        content.put("parts", Arrays.<Map<String, Object>>asList(ImmutableMap.<String, Object>of("text", "done")));
        candidate.put("content", content);
        candidates.add(candidate);
        body.put("candidates", candidates);

        stream.stream(JsonUtils.write(body));

        Assertions.assertTrue(notifyProcessCalled[0], "notifyProcess should be called when totalFinish and forceNotifierProcess");
        Assertions.assertFalse(Boolean.TRUE.equals(f.get(stream)), "forceNotifierProcess should be reset to false after notifyProcess");
    }

    /**
     * 复位后再次 stream() 带 totalFinish 时应走 notifySegment 而非 notifyProcess。
     */
    @Test
    public void testForceNotifierProcessResetThenNextStreamCallsNotifySegment() throws Exception {
        GoogleRequest request = new GoogleRequest();
        request.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        request.setTokenFirst(1024);
        request.setTokenBuffer(1024);
        TrackFunCallService t = EasyMock.createMock(TrackFunCallService.class);
        EasyMock.replay(t);

        final boolean[] notifySegmentCalled = {false};
        final boolean[] notifyProcessCalled = {false};
        GoogleStream stream = new GoogleStream(ProviderStreamConfig.<GoogleRequest>builder()
                .trackFunCallService(t)
                .tokenStatistic(ObjectBuilder.buildTokenStatistic())
                .mediaInlineService(ObjectBuilder.buildMediaInlineService())
                .notifierService(ObjectBuilder.buildActualNotifierManagerWithNothing())
                .providerReason(ObjectBuilder.getProviderReason())
                .signalStream(SignalStream.EMPTY)
                .historyStore(ObjectBuilder.buildHistoryStore())
                .namesService(ObjectBuilder.buildNamesService())
                .request(request)
                .build()) {
            @Override
            protected void notifyProcess() throws Exception {
                notifyProcessCalled[0] = true;
            }

            @Override
            protected void notifySegment() throws Exception {
                notifySegmentCalled[0] = true;
            }
        };

        Field f = GoogleStream.class.getDeclaredField("forceNotifierProcess");
        f.setAccessible(true);
        f.set(stream, true);

        Map<String, Object> body = new HashMap<>();
        List<Map<String, Object>> candidates = new ArrayList<>();
        Map<String, Object> candidate = new HashMap<>();
        candidate.put("finishReason", "STOP");
        Map<String, Object> content = new HashMap<>();
        content.put("role", "model");
        content.put("parts", Arrays.<Map<String, Object>>asList(ImmutableMap.<String, Object>of("text", "done")));
        candidate.put("content", content);
        candidates.add(candidate);
        body.put("candidates", candidates);
        stream.stream(JsonUtils.write(body));

        Assertions.assertTrue(notifyProcessCalled[0]);
        notifySegmentCalled[0] = false;
        notifyProcessCalled[0] = false;

        stream.stream(JsonUtils.write(body));

        Assertions.assertTrue(notifySegmentCalled[0], "second stream with totalFinish should call notifySegment after flag reset");
        Assertions.assertFalse(notifyProcessCalled[0], "second stream should not call notifyProcess");
    }
}
