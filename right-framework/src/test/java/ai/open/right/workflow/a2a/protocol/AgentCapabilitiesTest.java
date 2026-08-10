package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

public class AgentCapabilitiesTest {

    @Test
    void testGetSet() {
        AgentCapabilities capabilities = new AgentCapabilities();
        capabilities.setStateTransitionHistory(true);
        assertTrue(capabilities.getStateTransitionHistory());
        capabilities.setPushNotifications(true);
        assertTrue(capabilities.getPushNotifications());
        capabilities.setStreaming(true);
        assertTrue(capabilities.getStreaming());
        capabilities.setStateTransitionHistory(false);
        assertFalse(capabilities.getStateTransitionHistory());
        capabilities.setPushNotifications(false);
        assertFalse(capabilities.getPushNotifications());
        capabilities.setStreaming(false);
        assertFalse(capabilities.getStreaming());
    }
}