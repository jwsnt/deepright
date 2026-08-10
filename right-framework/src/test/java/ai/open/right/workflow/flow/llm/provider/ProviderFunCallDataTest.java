package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ProviderFunCallDataTest {

    @Test
    public void testSetGet() {
        ProviderFunCallData providerFunCallData = new ProviderFunCallData();
        Assert.assertNotNull(providerFunCallData.getResponses());
        Assert.assertNotNull(providerFunCallData.getRequests());
        List<LLMFunCallResponse> responses = new ArrayList<LLMFunCallResponse>();
        List<LLMFunCallRequest> requests = new ArrayList<LLMFunCallRequest>();
        providerFunCallData.setResponses(responses);
        providerFunCallData.setRequests(requests);
        Assert.assertEquals(responses, providerFunCallData.getResponses());
        Assert.assertEquals(requests, providerFunCallData.getRequests());
    }

    @Test
    public void testIsValid_falseWhenEmpty() {
        ProviderFunCallData data = new ProviderFunCallData();
        Assert.assertFalse(data.isValid());
    }

    @Test
    public void testIsValid_falseWhenOnlyResponses() {
        ProviderFunCallData data = new ProviderFunCallData();
        data.getResponses().add(EasyMock.createMock(LLMFunCallResponse.class));
        Assert.assertFalse(data.isValid());
    }

    @Test
    public void testIsValid_falseWhenOnlyRequests() {
        ProviderFunCallData data = new ProviderFunCallData();
        data.getRequests().add(EasyMock.createMock(LLMFunCallRequest.class));
        Assert.assertFalse(data.isValid());
    }

    @Test
    public void testAddFunCall_addsRequestAndResponseWhenBothValid() {
        LLMFunCallRequest req = EasyMock.createMock(LLMFunCallRequest.class);
        LLMFunCallResponse resp = EasyMock.createMock(LLMFunCallResponse.class);
        EasyMock.expect(req.isValid()).andReturn(true);
        EasyMock.expect(resp.isValid()).andReturn(true);
        EasyMock.replay(req, resp);
        ProviderFunCallData data = new ProviderFunCallData();
        data.addFunCall(req, resp);
        Assert.assertEquals(1, data.getRequests().size());
        Assert.assertEquals(1, data.getResponses().size());
        Assert.assertSame(req, data.getRequests().get(0));
        Assert.assertSame(resp, data.getResponses().get(0));
        Assert.assertTrue(data.isValid());
        EasyMock.verify(req, resp);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testAddFunCall_throwsWhenResponseInvalid() {
        LLMFunCallRequest req = EasyMock.createMock(LLMFunCallRequest.class);
        LLMFunCallResponse resp = EasyMock.createMock(LLMFunCallResponse.class);
        EasyMock.expect(resp.isValid()).andReturn(false);
        EasyMock.replay(resp);
        ProviderFunCallData data = new ProviderFunCallData();
        data.addFunCall(req, resp);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testAddFunCall_throwsWhenRequestInvalid() {
        LLMFunCallRequest req = EasyMock.createMock(LLMFunCallRequest.class);
        LLMFunCallResponse resp = EasyMock.createMock(LLMFunCallResponse.class);
        EasyMock.expect(resp.isValid()).andReturn(true);
        EasyMock.expect(req.isValid()).andReturn(false);
        EasyMock.replay(req, resp);
        ProviderFunCallData data = new ProviderFunCallData();
        data.addFunCall(req, resp);
    }

    @Test
    public void testGetMetadata_lazyInitializesMap() {
        ProviderFunCallData data = new ProviderFunCallData();
        Map<String, Object> m1 = data.getMetadata();
        Assert.assertNotNull(m1);
        Assert.assertTrue(m1.isEmpty());
        Assert.assertSame(m1, data.getMetadata());
    }

    @Test
    public void testPutMetadataAndGetMetadataTyped() throws Exception {
        ProviderFunCallData data = new ProviderFunCallData();
        data.putMetadata("k", 42);
        Assert.assertEquals(Integer.valueOf(42), data.getMetadata("k", Integer.class));
    }

    @Test
    public void testGetMetadataReturnsNullWhenMetadataEmpty() throws Exception {
        ProviderFunCallData data = new ProviderFunCallData();
        data.setMetadata(new HashMap<>());
        Assert.assertNull(data.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsNullWhenValueNull() throws Exception {
        ProviderFunCallData data = new ProviderFunCallData();
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("HELLO", null);
        data.setMetadata(metadata);
        Assert.assertNull(data.getMetadata("HELLO", String.class));
    }

    @Test
    public void testGetMetadataReturnsOriginalTypeWhenAssignable() throws Exception {
        ProviderFunCallData data = new ProviderFunCallData();
        Date date = new Date();
        data.putMetadata("HELLO", date);
        Assert.assertSame(date, data.getMetadata("HELLO", Date.class));
    }

    @Test
    public void testGetMetadataTransfersWhenTypeMismatch() throws Exception {
        ProviderFunCallData data = new ProviderFunCallData();
        data.putMetadata("CONFIG", Collections.singletonMap("provider", "PROVIDER"));
        LLMConfig config = data.getMetadata("CONFIG", LLMConfig.class);
        Assert.assertEquals("PROVIDER", config.getProvider());
    }

    @Test
    public void testToString_notNull() {
        ProviderFunCallData data = new ProviderFunCallData();
        Assert.assertNotNull(data.toString());
    }
}
