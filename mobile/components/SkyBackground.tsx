import { ImageBackground, StyleSheet, View, type ViewProps } from "react-native";

const skyImage = require("../assets/images/sky_clouds_bg.png");

export function SkyBackground({ style, children, ...rest }: ViewProps) {
  return (
    <View style={[styles.root, style]} {...rest}>
      <ImageBackground
        source={skyImage}
        style={styles.skyImage}
        imageStyle={styles.skyImageInner}
        resizeMode="cover"
      >
        <View style={styles.skyTint} pointerEvents="none" />
      </ImageBackground>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: "#5BB5F0",
  },
  skyImage: {
    ...StyleSheet.absoluteFill,
  },
  skyImageInner: {
    width: "100%",
    height: "100%",
  },
  skyTint: {
    ...StyleSheet.absoluteFill,
    backgroundColor: "rgba(91, 181, 240, 0.08)",
  },
});
