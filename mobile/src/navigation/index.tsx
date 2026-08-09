import React from 'react';
import { ActivityIndicator, View } from 'react-native';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { useAuth } from '../auth/AuthContext';
import { theme } from '../theme/colors';
import type { RootStackParamList, MainTabParamList } from './types';
import LoginScreen from '../screens/LoginScreen';
import ScanScreen from '../screens/ScanScreen';
import TasksScreen from '../screens/TasksScreen';
import MapScreen from '../screens/MapScreen';
import ProfileScreen from '../screens/ProfileScreen';
import CheckupWizardScreen from '../screens/CheckupWizardScreen';
import StreamScreen from '../screens/StreamScreen';
import ChangePasswordScreen from '../screens/ChangePasswordScreen';

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<MainTabParamList>();

// Ikon per tab.
const TAB_ICONS: Record<keyof MainTabParamList, keyof typeof Ionicons.glyphMap> = {
  Scan: 'scan-outline',
  Tasks: 'list-outline',
  Map: 'map-outline',
  Profile: 'person-outline',
};

function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.muted,
        tabBarStyle: {
          backgroundColor: theme.card,
          borderTopColor: theme.border,
        },
        tabBarIcon: ({ color, size }) => {
          const name = TAB_ICONS[route.name];
          return <Ionicons name={name} size={size} color={color} />;
        },
      })}
    >
      <Tab.Screen name="Scan" component={ScanScreen} options={{ title: 'Scan QR' }} />
      <Tab.Screen name="Tasks" component={TasksScreen} options={{ title: 'Tugas' }} />
      <Tab.Screen name="Map" component={MapScreen} options={{ title: 'Peta' }} />
      <Tab.Screen name="Profile" component={ProfileScreen} options={{ title: 'Profil' }} />
    </Tab.Navigator>
  );
}

export default function AppNavigator() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: theme.background }}>
        <ActivityIndicator size="large" color={theme.primary} />
      </View>
    );
  }

  return (
    <NavigationContainer>
      <Stack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: theme.card },
          headerTintColor: theme.text,
          contentStyle: { backgroundColor: theme.background },
        }}
      >
        {!isAuthenticated ? (
          <Stack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
        ) : (
          <>
            <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
            <Stack.Screen name="Checkup" component={CheckupWizardScreen} options={{ title: 'Pengecekan Lapangan' }} />
            <Stack.Screen name="Stream" component={StreamScreen} options={{ title: 'Live Stream' }} />
            <Stack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ title: 'Ganti Password' }} />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
