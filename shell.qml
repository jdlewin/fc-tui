// Quickshell frontend - runs on your Linux box, not on Mac.
// Reads feed.json written by backend/. Keeps memory low: 15 rows, text only.
// Run from repo root: quickshell -c .
import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

ShellRoot {
  // Auto-refresh backend every 2 min (saves Neynar credits + RAM).
  Timer {
    interval: 120000
    running: true
    repeat: true
    onTriggered: fetcher.running = true
  }

  Process {
    id: fetcher
    command: ["sh", "refresh.sh"]
    workingDirectory: Quickshell.shellDir
    // Run once at startup too.
    Component.onCompleted: fetcher.running = true
  }

  PanelWindow {
    anchors {
      top: true
      left: true
      right: true
    }
    implicitHeight: 420
    color: "#111111"

    ColumnLayout {
      anchors.fill: parent
      anchors.margins: 12
      spacing: 8

      Text {
        text: "Farcaster — following"
        color: "white"
        font.pixelSize: 16
        font.bold: true
      }

      Text {
        visible: !feedFile.loaded
        text: "Loading feed.json… run ./refresh.sh first"
        color: "#888888"
        font.pixelSize: 12
      }

      ListView {
        Layout.fillWidth: true
        Layout.fillHeight: true
        model: ScriptModel {
          values: feedCasts
        }
        spacing: 6
        delegate: Rectangle {
          width: ListView.view.width
          height: col.height + 12
          color: "#1c1c1c"
          radius: 8
          Column {
            id: col
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.margins: 8
            Text {
              width: parent.width
              text: "@" + (modelData.author?.username ?? "unknown")
              color: "#8b5cf6"
              font.pixelSize: 13
              font.bold: true
              elide: Text.ElideRight
            }
            Text {
              width: parent.width
              text: modelData.text ?? ""
              color: "#e5e5e5"
              font.pixelSize: 13
              wrapMode: Text.Wrap
              maximumLineCount: 4
              elide: Text.ElideRight
            }
          }
        }
      }

      Text {
        text: "Auto-refresh every 2 min via refresh.sh"
        color: "#666666"
        font.pixelSize: 11
      }
    }
  }

  FileView {
    id: feedFile
    path: Qt.resolvedUrl("./feed.json")
    watchChanges: true
    blockLoading: true
    onFileChanged: this.reload()
  }

  readonly property var feedCasts: {
    if (!feedFile.loaded)
      return [];
    try {
      const data = JSON.parse(feedFile.text());
      return (data.casts ?? []).slice(0, 15);
    } catch (e) {
      return [];
    }
  }
}
