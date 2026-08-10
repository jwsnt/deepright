package ai.open.right.workflow.a2a.protocol;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

import java.util.List;

public class AgentCardTest {

    @Test
    void testGetSet() {
        AgentCard card = new AgentCard();
        AgentCapabilities capabilities = new AgentCapabilities();
        card.setCapabilities(capabilities);
        assertSame(capabilities, card.getCapabilities());
        List<String> outputModes = List.of("test");
        card.setDefaultOutputModes(outputModes);
        assertSame(outputModes, card.getDefaultOutputModes());
        List<String> inputModes = List.of("test2");
        card.setDefaultInputModes(inputModes);
        assertSame(inputModes, card.getDefaultInputModes());
        card.setPreferredTransport("TEST");
        assertEquals("TEST", card.getPreferredTransport());
        card.setProtocolVersion("1.0");
        assertEquals("1.0", card.getProtocolVersion());
        card.setDescription("");
        assertEquals("a2a server", card.getDescription());
        card.setDescription("desc");
        assertEquals("desc", card.getDescription());
        List<AgentSkill> skills = List.of(new AgentSkill());
        card.setSkills(skills);
        assertSame(skills, card.getSkills());
        assertTrue(card.hasSkill());
        card.setSkills(null);
        assertFalse(card.hasSkill());
        card.setVersion("2.0");
        assertEquals("2.0", card.getVersion());
        card.setName("name");
        assertEquals("name", card.getName());
        card.setUrl("url");
        assertEquals("url", card.getUrl());
        assertEquals(List.of("application/json"), AgentCard.MODELS);
    }
}