package ai.deepright.lang;

import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.net.URLConnection;
import java.util.List;
import java.util.Locale;
import java.util.Properties;
import java.util.ResourceBundle;

public class XmlResourceControl extends ResourceBundle.Control {

    public static final String FORMAT_XML = "xml";

    @Override
    public List<String> getFormats(String baseName) {
        Assert.notNull(baseName, "The base name must not be empty");
        return List.of(XmlResourceControl.FORMAT_XML);
    }

    @Override
    public ResourceBundle newBundle(String baseName, Locale locale, String format, ClassLoader loader, boolean reload) throws IllegalAccessException, InstantiationException, IOException {
        Assert.notNull(baseName, "The base name must not be empty");
        Assert.notNull(locale, "The locale must not be empty");
        Assert.notNull(format, "The format must not be empty");
        Assert.notNull(loader, "The loader must not be empty");
        Assert.isTrue(XmlResourceControl.FORMAT_XML.equals(format), "The format can not be support:" + format);
        String resourceName = this.toResourceName(this.toBundleName(baseName, locale), XmlResourceControl.FORMAT_XML);
        ResourceBundle bundle = null;
        InputStream stream = null;
        if (reload) {
            URL url = loader.getResource(resourceName);
            if (url == null) {
                return null;
            }
            URLConnection connection = url.openConnection();
            connection.setUseCaches(false);
            stream = connection.getInputStream();
        } else {
            stream = loader.getResourceAsStream(resourceName);
        }
        if (stream == null) {
            return null;
        }
        try (InputStream input = new BufferedInputStream(stream)) {
            Properties properties = new Properties();
            properties.loadFromXML(input);
            bundle = new XmlResourceBundle(properties);
        }
        return bundle;
    }
}
