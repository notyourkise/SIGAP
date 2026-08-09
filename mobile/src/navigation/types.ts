// Tipe navigasi aplikasi SIGAP Mobile.
import type { NavigatorScreenParams } from '@react-navigation/native';
import type { CCTV } from '../api/types';

// Stack utama (di luar tab) — untuk login dan layar penuh.
export type RootStackParamList = {
  Login: undefined;
  Main: NavigatorScreenParams<MainTabParamList> | undefined;
  Checkup: { cctv: CCTV } | undefined;
  Stream: { cctv: CCTV } | undefined;
  ChangePassword: undefined;
};

// Bottom tabs utama untuk teknisi di lapangan.
export type MainTabParamList = {
  Scan: undefined;
  Tasks: undefined;
  Map: undefined;
  Profile: undefined;
};
