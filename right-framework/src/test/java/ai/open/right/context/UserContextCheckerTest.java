package ai.open.right.context;

import org.junit.Assert;
import org.junit.Test;

public class UserContextCheckerTest {

    @Test
    public void testSuccess() {
        UserContext user = UserContext.builder().build();
        user.setLanguage("La");
        user.setDevice("De");
        user.setRegion("Re");
        user.setSystem("Sy");
        user.setBrand("Br");
        user.setToken("To");
        user.setModel("Mo");
        Assert.assertEquals(user.getToken(), "To");
        Assert.assertEquals(user.getBrand(), "Br");
        Assert.assertEquals(user.getModel(), "Mo");
        Assert.assertEquals(user.getDevice(), "De");
        Assert.assertEquals(user.getRegion(), "Re");
        Assert.assertEquals(user.getSystem(), "Sy");
        UserContext.UserContextChecker.check(user);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutSystem() {
        UserContext user = UserContext.builder()
                .language("La")
                .region("Re")
                .device("De")
                .brand("Br")
                .token("To")
                .build();
        UserContext.UserContextChecker.check(user);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutRegion() {
        UserContext user = UserContext.builder()
                .language("La")
                .system("Sy")
                .device("De")
                .brand("Br")
                .token("To")
                .build();
        UserContext.UserContextChecker.check(user);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutDevice() {
        UserContext user = UserContext.builder()
                .language("La")
                .system("Sy")
                .region("Re")
                .token("To")
                .brand("Br")
                .build();
        UserContext.UserContextChecker.check(user);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutBrand() {
        UserContext user = UserContext.builder()
                .language("La")
                .device("De")
                .system("Sy")
                .region("Re")
                .token("To")
                .build();
        UserContext.UserContextChecker.check(user);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testWithOutModel() {
        UserContext user = UserContext.builder()
                .language("La")
                .region("Re")
                .device("De")
                .system("Sy")
                .brand("Br")
                .token("To")
                .build();
        UserContext.UserContextChecker.check(user);
    }

    @Test
    public void testWithOutToken() {
        // Success
        UserContext user = UserContext.builder()
                .language("La")
                .region("Re")
                .system("Sy")
                .device("De")
                .brand("Br")
                .model("Mo")
                .build();
        UserContext.UserContextChecker.check(user);
    }
}
