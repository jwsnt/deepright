package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.reason.ProviderReason;
import ai.open.right.workflow.flow.llm.signal.SignalStream;
import org.easymock.EasyMock;
import org.junit.Assert;
import org.junit.Test;

public class ProviderQueryServiceTest {

    @Test
    public void testQuery1() throws Exception {
        ProviderRequestService providerRequestService = EasyMock.createMock(ProviderRequestService.class);
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(providerRequestService.config(llmConfig, llmQuery)).andReturn(providerRequest).anyTimes();
        ProviderRouter providerRouter = EasyMock.createMock(ProviderRouter.class);
        ProviderStream providerStream = EasyMock.createMock(ProviderStream.class);
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        providerRouter.route(providerRequest, llmConfig, providerStream);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(providerRequestService, providerRouter, providerStream, signalStream, providerRequest);
        ProviderQueryService providerQueryService = new ProviderQueryService() {

            @Override
            protected ProviderStream stream(SignalStream signalStream, ProviderRequest r) {
                return providerStream;
            }

            @Override
            protected ProviderRequestService request() {
                return providerRequestService;
            }

            @Override
            protected ProviderRouter router() {
                return providerRouter;
            }
        };
        ProviderReason reason = EasyMock.createMock(ProviderReason.class);
        providerQueryService.setProviderReason(reason);
        Assert.assertEquals(reason, providerQueryService.getProviderReason());
        providerQueryService.query(llmQuery, llmConfig, signalStream);
        EasyMock.verify(providerRequestService, providerRouter);
    }

    @Test
    public void testQuery2() throws Exception {
        ProviderRequestService providerRequestService = EasyMock.createMock(ProviderRequestService.class);
        ProviderRequest providerRequest = EasyMock.createMock(ProviderRequest.class);
        LLMConfig llmConfig = new LLMConfig();
        LLMQuery llmQuery = ObjectBuilder.buildLLMQuery();
        EasyMock.expect(providerRequestService.config(llmConfig, llmQuery)).andReturn(providerRequest).anyTimes();
        ProviderRouter providerRouter = EasyMock.createMock(ProviderRouter.class);
        ProviderStream providerStream = EasyMock.createMock(ProviderStream.class);
        SignalStream signalStream = EasyMock.createMock(SignalStream.class);
        providerRouter.route(providerRequest, llmConfig, providerStream);
        EasyMock.expectLastCall().anyTimes();
        EasyMock.replay(providerRequestService, providerRouter, providerStream, signalStream, providerRequest);
        ProviderQueryService providerQueryService = new ProviderQueryService() {

            @Override
            protected ProviderStream stream(SignalStream signalStream, ProviderRequest r) {
                return providerStream;
            }

            @Override
            protected ProviderRequestService request() {
                return providerRequestService;
            }

            @Override
            protected ProviderRouter router() {
                return providerRouter;
            }
        };
        providerQueryService.query(llmQuery, llmConfig);
        EasyMock.verify(providerRequestService, providerRouter);
    }
}
