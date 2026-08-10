package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import ai.open.right.workflow.flow.llm.store.history.HistoryStore;
import ai.open.right.workflow.flow.media.MediaInlineService;
import ai.open.right.workflow.flow.track.TrackFunCallService;
import ai.open.right.workflow.notify.NotifierService;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class ProviderStreamConfigTest {

    private static ProviderStreamConfig.ProviderStreamConfigBuilder<ProviderRequest> baseBuilder() {
        return ProviderStreamConfig.<ProviderRequest>builder()
                .providerStorePolicy(EasyMock.createMock(ProviderStorePolicy.class))
                .trackFunCallService(EasyMock.createMock(TrackFunCallService.class))
                .mediaInlineService(EasyMock.createMock(MediaInlineService.class))
                .notifierService(EasyMock.createMock(NotifierService.class))
                .signalStream(EasyMock.createMock(SignalStream.class))
                .historyStore(EasyMock.createMock(HistoryStore.class))
                .namesService(EasyMock.createMock(NamesService.class))
                .request(new ProviderRequest());
    }

    @Test
    public void testCheck_success_returnsThis() throws Exception {
        ProviderStreamConfig<ProviderRequest> config = baseBuilder().build();
        Assert.assertSame(config, config.check());
    }

    @Test
    public void testCheck_optionalFieldsNull_succeeds() throws Exception {
        ProviderStreamConfig<ProviderRequest> config = baseBuilder()
                .tokenStatistic(null)
                .providerReason(null)
                .extension(null)
                .build();
        config.check();
    }

    @Test
    public void testCheck_extensionPopulated_succeeds() throws Exception {
        Map<String, Object> ext = new HashMap<>();
        ext.put("k", 1);
        ProviderStreamConfig<ProviderRequest> config = baseBuilder()
                .extension(Collections.unmodifiableMap(ext))
                .build();
        Assert.assertEquals(1, config.getExtension().get("k"));
        config.check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullNotifierService() throws Exception {
        baseBuilder().notifierService(null).build().check();
    }

    @Test
    public void testCheck_nullSignalStream() throws Exception {
        baseBuilder().signalStream(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullHistoryStore() throws Exception {
        baseBuilder().historyStore(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullNamesService() throws Exception {
        baseBuilder().namesService(null).build().check();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheck_nullRequest() throws Exception {
        baseBuilder().request(null).build().check();
    }

    @Test
    public void testCheck_nullRequest_message() throws Exception {
        try {
            baseBuilder().request(null).build().check();
            Assert.fail();
        } catch (IllegalArgumentException e) {
            Assert.assertNotNull(e.getMessage());
            Assert.assertTrue(e.getMessage().toLowerCase().contains("request"));
        }
    }
}
